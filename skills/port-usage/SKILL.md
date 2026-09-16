---
name: port-usage
description: >
    Use when allocating, moving, auditing, or debugging a TCP port for a local
    service — picking the next free number, deciding whether a service is public
    or internal, publishing a compose `ports:` entry, pointing a Cloudflare Tunnel
    ingress at it, registering a health endpoint, or resolving "address already in
    use". Enforces the 83xx (public) / 85xx (internal) segmentation. Triggers on:
    "which port should this use", "allocate a port", "port conflict", "address
    already in use", "expose this service", "分配 port", "埠號", "port 衝突",
    "改 port", "對外開放".
version: "1.0.0"
allowed-tools: Read, Edit, Write, Bash, Grep
user-invocable: true
disable-model-invocation: false
effort: medium
context: fork
metadata:
    type: technique
    platforms: [macos, linux]
---

# Port 配置 (Port Usage)

## 概要 (Overview)

一個 port 號碼在這個工作區裡代表三件事，缺一不可：

1. `身分 (identity)`：一個號碼永遠只屬於一個 service，且在 host 側與 container 側是同一個數字。
2. `曝光面 (exposure)`：號碼落在哪一段，就宣告了誰連得到它——`83xx` 對外、`85xx` 對內。
3. `契約 (contract)`：同一個數字必須同時出現在 compose 的 `ports:`、來源 repo 的 default port、
   以及（若對外）tunnel 的 ingress。三處有一處不同步，服務就是壞的，而且`不會報錯`。

分段的目的不是編號整齊，是`看號碼就知道該不該綁 loopback`。稽核曝光面時讀 `ports:` 的
host IP 前綴，不讀應用程式的 bind 設定——後者在 container 內是 container 的 loopback，
永遠不是你以為的那個。

## 分段 (Segments)

| 段別 | 範圍 | 用途 | 發佈方式 |
| --- | --- | --- | --- |
| `public` | `8300`–`8399` | 有 tunnel ingress，或 LAN 上的人會直接開的 HTTP 端點 | `"<port>:<port>"` |
| `internal` | `8500`–`8599` | 只被同機其他 service、本機 CLI、SSH tunnel 消費的（含`未認證的管理介面`） | `"127.0.0.1:<port>:<port>"` |
| `well-known` | 上游既定 | 第三方 image：`mysql` 3306、`postgres` 5432、`registry` 5000、`redis` 6379… | 照上游慣例，`不`改號 |
| `observability` | 既定 | `inf` 的 LGTM 棧（3000/3100/3200/8428/9009/9090/9093…） | 見 `inf/CLAUDE.md` 的埠號表 |

判準是`誰連得到它`，不是程式跑在哪、也不是它是不是 HTTP。
同一個 service 有兩種介面時`兩段各取一個`（intake API 在 `83xx`、admin console 在 `85xx`），
因為曝光面屬於 port，不屬於 service。

## 使用時機 (When to Use)

- 新服務要選號，或既有服務要改號
- 判斷某個服務該不該對外，以及該綁 `0.0.0.0` 還是 `127.0.0.1`
- `address already in use` / `port is already allocated`
- 稽核：哪些 port 對外開著、哪些 ingress 指向不存在的服務
- 服務起來了但連不到，或 tunnel hostname 回 502

不適用：容器內部的服務間通訊（同一個 compose network 用 service 名稱直連，
`不需要`也`不應該` publish 到 host）。

## Hard Rules

| # | 規則 | 違反時 |
| --- | --- | --- |
| 1 | host port `≡` container port | 改成相同數字；轉譯層會製造「這個號碼在哪一側」的長期歧義 |
| 2 | public 取 `83xx`、internal 取 `85xx` | 判定曝光面後改號，並同步三處 |
| 3 | internal 一律 `127.0.0.1:` 前綴 | 補上前綴；container 內綁 loopback `不是`隔離，是自廢 |
| 4 | 號碼`不回收` | 下線的號碼留空。重用會讓握著舊號碼的 client 靜默連到別的服務 |
| 5 | 一個號碼只屬於一個 service，`沒有例外` | 改號。`互斥執行`不構成理由——「只能開一個」是一句靠人記得的約定，而號碼衝突要到啟動失敗才會被發現 |
| 6 | 第三方 image 不套分段 | 沿用 well-known port；改號的成本落在每一個 client 的設定上 |
| 7 | 只被其他容器消費的 port `不 publish` | 從 `ports:` 移到同網路的 service 名稱直連 |
| 8 | 沒有認證的介面`永遠` internal | 沒有例外。認證不是「之後再加」的項目 |

## 配置流程 (Allocation Workflow)

### 1. 判定曝光面 (Classify)

依序問，第一個「是」就決定答案：

- 會不會有 Cloudflare Tunnel 的 `hostname` 指向它？→ `public`
- LAN 上的人（手機、另一台機器、瀏覽器）會不會直接開它？→ `public`
- 它有認證嗎？沒有 → `internal`，無論原本打算怎麼用
- 其餘（另一個 service、本機 CLI、SSH tunnel 消費）→ `internal`

### 2. 掃描已用號碼 (Scan)

`三個來源都要看`——compose 不是唯一的配置者，pm2 管的 host 程序也佔號碼：

```bash
# a) 所有 compose 的 host 側 port（在基礎設施 repo 的根目錄執行）
grep -rhoE '"([0-9.]+:)?[0-9]{2,5}:[0-9]{2,5}"' \
    $(git rev-parse --show-toplevel)/**/docker-compose*.yml 2>/dev/null \
  | tr -d '"' | awk -F: '{print $(NF-1)}' | sort -nu

# b) 此刻真的在聽的（含常駐 host 程序、開發中的 dev server）
ss -tlnH | awk '{print $4}' | sed 's/.*://' | sort -nu      # linux
lsof -nP -iTCP -sTCP:LISTEN | awk 'NR>1{print $9}' | sed 's/.*://' | sort -nu  # macos

# c) 埠號登記表（含已下線但保留的號碼）：本 repo 的 config/default_settings.json 是 embed 預設值，
#    host 上若有 ~/.config/port/settings.json，它的 ports 陣列會整個蓋掉預設——兩份都要看
jq -r '.ports[] | "\(.port)\t\(.name)"' "${PORT_REGISTRY:-$HOME/.config/port/settings.json}" | sort -n
```

`路徑不寫死`：(a) 用 `git rev-parse --show-toplevel` 從當前 repo 定位，
(c) 預設讀 host 的 `~/.config/port/settings.json`，`PORT_REGISTRY` 可覆寫。
這個技能是`規則`，不是某台機器的地圖——換工作區只換這兩個入口。

`只有 (a) 是宣告，(b) 是現況，(c) 是歷史`。取號要避開三者的聯集。

### 3. 取號 (Pick)

取該段`最小的未使用號碼`。不跳號、不按專案分群、不預留區塊——預留的區塊
最後總是被別的東西佔走，然後沒有人記得為什麼那裡有個洞。

### 4. 同步三處 (Wire)

```yaml
# 1. compose 的 ports:（曝光面在這裡決定）
ports:
  - "8310:8310"              # public
  - "127.0.0.1:8510:8510"    # internal
```

```yaml
# 2. 來源 repo 的 default port —— 同一個數字，不是「差不多」
environment:
    PORT: 8310
```

```yaml
# 3. 只有 public 才做：tunnel ingress（catch-all 永遠留最後）
- hostname: <name>.<your-domain>
  service: http://localhost:8310
```

改號時`三處一起改`。只改 compose 會得到一個對得上 host 卻對不上 ingress 的服務，
`docker ps` 完全正常，只有使用者看到 502。

### 5. 登記與驗證 (Register & Verify)

```bash
# 登記到 port 工具，有 HTTP 健康端點就寫 URL，資料庫類寫 "tcp"，自簽憑證加 "insecure": true
#   repo 的 config/default_settings.json（embed 預設）與 host 的 ~/.config/port/settings.json 兩份都要改
#   { "port": 8310, "name": "<service>", "health": "http://localhost:8310/health" }

docker compose up -d <service>
ss -tlnH | grep ':8310'        # 確認綁定位址：0.0.0.0 vs 127.0.0.1
curl -fsS localhost:8310/health
port health                     # 全域關卡，任一 FAIL 以非零碼結束
```

`internal 的驗收條件是「從別台機器連不到」`，不只是本機連得到：
`ss` 那行看到 `127.0.0.1:8510` 才算過，看到 `0.0.0.0:8510` 就是沒綁對。

## 稽核 (Audit)

```bash
root=$(git rev-parse --show-toplevel)

# 1. 對外開著的 port（沒有 host IP 前綴的 publish）
grep -rnE '^\s+- "[0-9]{2,5}:[0-9]{2,5}"' "$root" --include='docker-compose*.yml'

# 2. host/container 兩側不一致
grep -rhoE '"[0-9]{2,5}:[0-9]{2,5}"' "$root" --include='docker-compose*.yml' \
  | tr -d '"' | awk -F: '$1!=$2 {print "MISMATCH " $0}'

# 3. ingress 指向沒有人在聽的號碼（$TUNNEL_CONFIG = tunnel 的 ingress 設定檔）
grep -oE 'localhost:[0-9]+' "$TUNNEL_CONFIG" | cut -d: -f2 | sort -u \
  | while read -r p; do ss -tlnH | grep -q ":$p " || echo "DEAD ingress -> $p"; done
```

三條的判準都在`設定檔`，不在應用程式——這是刻意的。應用程式的 bind 設定
說不出「LAN 上的人連不連得到」，`ports:` 的 host 側說得出。

## 反模式 (Anti-patterns)

| 反模式 | 為什麼壞 | 改法 |
| --- | --- | --- |
| `"8080:8310"` | 兩側不同，之後每次 debug 都要先問「這是哪一側」 | 兩側同號 |
| 應用程式綁 `127.0.0.1`，compose 發佈 `0.0.0.0` | container 內的 loopback 連 host 都到不了，服務直接連不上 | 應用綁 `0.0.0.0`，用 `ports:` 的 host 側限制來源 |
| 管理介面「先開著，之後加認證」 | 「之後」不會來，而 LAN 一直開著 | 一開始就放 `85xx` + loopback |
| service 間通訊也 publish 到 host | 多一個對外開口，換不到任何東西 | 同 network 用 service 名稱直連 |
| 舊服務下線後把號碼給新服務 | 舊 client 連得上、但拿到完全不同的東西，且不報錯 | 號碼留空 |
| 用 `ufw` 擋 docker publish 的 port | docker 直接寫 `DOCKER-USER` 之前的 nat 規則，`ufw deny` 擋不住 | 綁定位址寫在 `ports:` 的 host 側 |

## 疑難排解 (Troubleshooting)

| 症狀 | 診斷 | 處置 |
| --- | --- | --- |
| `address already in use` | `ss -tlnp \| grep ':<port>'`（macOS：`lsof -nP -iTCP:<port> -sTCP:LISTEN`） | 是自己的殘留 → `port kill <port>`；是別的服務 → 換號 |
| `port is already allocated`（compose） | `docker ps --format '{{.Names}}\t{{.Ports}}' \| grep <port>` | 舊 container 沒收乾淨 → `docker compose down` 再起 |
| 本機連得到、別台連不到 | `ss -tlnH \| grep ':<port>'` | 綁在 `127.0.0.1` → 若確實該對外，移到 `83xx` 並移除前綴 |
| tunnel hostname 回 502 | ingress 的號碼 vs `ports:` 的號碼 | 兩者對不上 → 同步；服務沒起來 → `docker compose up -d` |
| `port health` 有 FAIL 但服務正常 | 登記的 health URL | 端點路徑或 scheme 寫錯 → 改登記表，不是改服務 |

## 參考 (References)

本技能`只擁有規則`，不擁有任何配置表。每一項的事實來源都在使用它的 repo 裡：

| 事實 | 去哪裡找 |
| --- | --- |
| 分段規則的採用與例外 | 該 repo `CLAUDE.md` 的「關鍵決策」 |
| 誰佔了哪個號碼 | 各 `docker-compose*.yml` 的 `ports:`——`唯一`事實來源 |
| 觀測性元件的既定埠號 | `inf/CLAUDE.md` 的埠號表 |
| 健康端點登記 | 本 repo `config/default_settings.json` + host `~/.config/port/settings.json` |
| Tunnel ingress 的機制 | `inf/docs/cloudflare-tunnel.md` |
| 容器化與 deployment.yml 的契約 | `inf` 的 `inf-spec` skill |
