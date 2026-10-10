# A股股票交易撮合系统

基于 **Go + PostgreSQL 13 + Redis** 后端与 **Vue3 + Element Plus** 前端的模拟 A 股撮合交易系统，支持管理员发行股票、设置涨跌幅与开闭市规则，普通用户限价买卖，前端交易大盘展示当日汇总图表。

## 功能特性

- **JWT 鉴权**：注册 / 登录，管理员与普通用户角色隔离
- **股票发行**：管理员可发行股票，设置单日涨跌幅限制、停牌
- **开闭市规则**：上午/下午交易时段、交易日、强制开市/休市
- **限价撮合**：价格优先、时间优先；成交价取对手方价格
- **资金与持仓**：买入冻结资金，卖出冻结持仓，支持撤单退回
- **交易大盘**：行情列表、分时成交量/额、涨跌分布（ECharts）
- **五档盘口**：买卖挂单深度展示

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.21、Gin、GORM、golang-jwt |
| 数据库 | PostgreSQL 13 |
| 缓存 | Redis（compose 使用 redis-stack-server，行情汇总等） |
| 前端 | Vue 3、Vite、Element Plus、Pinia、ECharts |
| 部署 | Docker Compose、Nginx |

## 端口

| 服务 | 端口 |
|------|------|
| 前端 | **6173** |
| 后端 API | **10020** |
| PostgreSQL / Redis | 不对外暴露，仅 Docker 内网访问 |

## 快速启动（Docker Compose）

```bash
# 构建并启动全部服务
make up

# 或
docker compose up -d --build
```

启动后访问：

- 前端：http://localhost:6173
- 后端健康检查：http://localhost:10020/health

### 默认账号

| 角色 | 用户名 | 密码 | 说明 |
|------|--------|------|------|
| 管理员 | `admin` | `admin123` | 发行股票、规则配置 |
| 演示卖方 | `trader` | `trader123` | 已持有演示股票，可挂卖单 |
| 普通用户 | 自行注册 | ≥6 位 | 初始资金 100 万 |

演示环境默认 **强制开市**，便于任意时间测试撮合。

### 常用 Make 命令

```bash
make help          # 查看命令说明
make up            # 构建并后台启动
make down          # 停止服务
make logs          # 查看日志
make rebuild       # 无缓存重建
make tidy          # 整理 Go 依赖
make backend-run   # 本地跑后端（需本机 PG/Redis）
make frontend-run  # 本地跑前端开发服
```

## 本地开发

### 依赖

- Go 1.21+
- Node.js 18+
- PostgreSQL 13
- Redis

### 后端

```bash
# Postgres / Redis 默认不映射宿主机端口，本地联调需自备实例，或临时在 compose 中加 ports
cd backend
go mod tidy
export DATABASE_URL='postgres://stock:stock123@localhost:5432/stock_matching?sslmode=disable'
export REDIS_ADDR=localhost:6379
export PORT=10020
go run ./cmd/server
```

### 前端

```bash
cd frontend
npm install
npm run dev
# http://localhost:6173 ，/api 代理到 :10020
```

## API 概览

前缀：`/api/v1`，需鉴权接口在 Header 携带：`Authorization: Bearer <token>`

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/auth/register` | 注册 |
| POST | `/auth/login` | 登录 |
| GET | `/auth/me` | 当前用户 |
| GET | `/stocks` | 股票列表 |
| GET | `/stocks/quotes` | 行情 |
| GET | `/market/status` | 开市状态 |
| GET | `/admin/market/summary` | 当日交易汇总（仅管理员；交易大盘） |
| GET | `/stocks/:id/orderbook` | 五档盘口 |
| GET | `/stocks/:id/kline?interval=1d&limit=120` | K 线（1m/5m/15m/1d） |
| GET | `/market/stream` | SSE 实时行情（tick：报价/成交/K线） |
| POST | `/orders` | 下单 |
| DELETE | `/orders/:id` | 撤单 |
| GET | `/orders` | 我的委托 |
| GET | `/trades` | 我的成交 |
| GET | `/holdings` | 我的持仓 |
| POST | `/admin/stocks` | 发行股票（管理员） |
| PUT | `/admin/stocks/:id` | 更新涨跌幅/状态（管理员） |
| GET/PUT | `/admin/market/rules` | 查询/更新开闭市规则（管理员） |

### 下单示例

```bash
TOKEN=... # 登录后获取

curl -X POST http://localhost:10020/api/v1/orders \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"stock_id":"<uuid>","side":"buy","price":10.50,"quantity":100}'
```

## 实时行情（SSE）

前端通过 `EventSource` 连接 `/api/v1/market/stream`（可选 `?stock_id=` 过滤）：

- `connected` / `heartbeat`：连接与保活
- `tick`：撮合/撤单后推送，包含最新 `quote`、本次 `trades`、各周期最新 `klines`

历史 K 线仍用 REST 拉取；当前蜡烛由 SSE 增量更新。

## 撮合规则说明

1. 仅在开市时段（或强制开市）接受委托
2. 委托价格须在 `[昨收×(1-涨跌幅%), 昨收×(1+涨跌幅%)]` 内
3. 数量须为 **100 股整数倍**
4. 买方按委托价冻结资金；卖方冻结持仓
5. 撮合：买价 ≥ 卖价时可成交，**价格优先、同价时间优先**
6. 成交价取**对手方（先挂单）价格**；买单差价退回可用资金
7. 撤单释放未成交部分的冻结资金/持仓

## 目录结构

```
.
├── backend/                 # Go API 服务
│   ├── cmd/server/          # 入口
│   ├── internal/
│   │   ├── cache/           # Redis
│   │   ├── config/
│   │   ├── database/        # Postgres + 种子数据
│   │   ├── handlers/        # HTTP 路由
│   │   ├── middleware/      # JWT / CORS
│   │   ├── models/
│   │   └── services/        # 鉴权、股票、市场、撮合
│   └── Dockerfile
├── frontend/                # Vue3 前端
│   ├── src/
│   │   ├── views/           # 大盘 / 交易 / 委托 / 管理
│   │   ├── stores/
│   │   └── api/
│   ├── nginx.conf
│   └── Dockerfile
├── docker-compose.yml
├── Makefile
└── README.md
```

## 环境变量（后端）

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `PORT` | `10020` | HTTP 端口 |
| `DATABASE_URL` | 见 compose | Postgres 连接串 |
| `REDIS_ADDR` | `localhost:6379` | Redis 地址 |
| `JWT_SECRET` | 内置演示值 | 生产务必修改 |
| `JWT_EXPIRE_HOURS` | `24` | Token 有效期 |

## 许可证

MIT
