# 本地电气台账系统

一个**本地、无密码、单机运行**的电气设备台账管理系统。

- 总表 8 列：配电室、配电柜、名称、型号、厂家、数量、备注、图片
- 图片最多 9 张/条，以 uuid 重命名存储于 `./data/annex`
- 左侧 tab：配电室 / 配电柜 / 元器件；配电室、配电柜支持名称 / 图片 / 备注，下拉菜单 + 弹窗新建，元器件弹窗新建
- 支持按配电室、配电柜筛选，按名称 / 型号搜索（后端过滤 + 分页）
- 附件引用自动计数，可随时重算，孤儿图片可一键清理
- 数据持久化于 `./data/app.db`，重启不丢失；数据/附件目录启动时自动创建

## 技术栈

| 层 | 技术 |
| --- | --- |
| 前端 | Vue 3 + Vite + Vue Router + Naive UI + TypeScript |
| 后端 | Go + Gin + sqlite3（`modernc.org/sqlite`，纯 Go 实现） |
| 存储 | SQLite（`./data/app.db`）+ 附件目录（`./data/annex`） |

生产模式下前端构建产物由后端 `go:embed` 托管，最终是**单文件二进制**，单端口访问。SQLite 采用纯 Go 驱动，`CGO_ENABLED=0` 即可交叉编译出**静态单文件**，支持 Linux x64 / Linux arm64 / Windows x64，无需安装任何依赖或交叉编译器。

## 快速开始

前置：Node.js 18+、pnpm。Go 若未安装，用 `make bootstrap` 自动安装最新稳定版。

```bash
make run     # 构建前端 + 后端并启动
# 打开 http://127.0.0.1:5288
```

- 端口：环境变量 `PORT`（默认 5288）
- 数据目录：环境变量 `DATA_DIR`（默认 `./data`）

### 多平台构建

```bash
make build-all     # 产出 bin/ledger-linux-amd64、ledger-linux-arm64、ledger-windows-amd64.exe
make build-linux-amd64    # 单独构建某一平台
make build-linux-arm64
make build-windows-amd64
```

每个产物均为内嵌前端页面的静态单文件，拷到对应机器直接运行（Windows 运行 `.exe`）即可。

### 体积说明与优化

构建产物约 **19MB/平台**（Windows 略大），主要构成：
- Go 运行时 + 标准库（基础体积，不可裁剪）
- Gin v1.12 无条件携带 HTTP/3（quic-go，拖入大量 crypto/x509 代码）与 BSON/protobuf 绑定 —— 本项目用不到，但**为不改源码只加构建参数**而保留
- `modernc.org/sqlite` 纯 Go 实现（比 CGO 版大，但换来三平台免交叉编译）

已通过**纯构建参数**优化（不改任何源码/依赖）：

| 参数 | 作用 | 节省 |
| --- | --- | --- |
| `-tags nomsgpack` | 去掉 gin 用不到的 msgpack 绑定（连带移除 ugorji codec） | ~6MB |
| `-trimpath` | 裁剪编译路径 | 少量 |
| `-ldflags "-s -w"` | 去掉 DWARF 调试信息与符号表 | ~2MB |
| `-buildvcs=false` | 不嵌入 VCS 信息 | 少量 |

> 对比：同样代码不加参数约 25.8MB → 加参数后约 19MB。

**可选：UPX 二次压缩**（`make upx` 或 CI 手动触发勾选 compress）可再压到 **~7.6MB**（-60%）。注意：UPX 压缩后的 Windows 版可能被杀毒软件误报，且启动时需解压（略增内存/启动耗时），请按需使用。

### 开发模式（两个终端）

```bash
# 终端 1
cd backend && go run ./cmd/server        # API :5288

# 终端 2
cd frontend && pnpm dev                   # 页面 http://localhost:5173，/api 已代理到 :5288
```

### 测试

```bash
make test     # 后端 go test ./... + 前端 vitest run
```

## 数据模型

- `rooms` 配电室（name 唯一）
- `cabinets` 配电柜（所属 room_id，同房间内 name 唯一）
- `equipment` 台账记录（room_id 必填，cabinet_id 可空，数量非负）
- `annex` 附件表：`uuid`（磁盘文件名）、`original_name`、`ext`、`mime_type`、`size`、`ref_count`（引用次数）
- `annex_ref` 引用映射表：`annex_id` → `target_type`(room/cabinet/equipment) + `target_id` + `position`（图片顺序）

**引用计数机制**：`annex.ref_count` 为冗余列，由 `annex_ref` 聚合而来。增删改引用时会对受影响附件即时重算；服务**启动时**自动全量重算；顶栏也提供「重算引用次数」按钮（`POST /api/annex/recompute`）。`ref_count=0` 的附件为孤儿（如上传后取消表单），顶栏「清理未引用图片」（`POST /api/annex/cleanup`）可删除其数据库记录与磁盘文件。

## API 摘要

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET/POST | `/api/rooms` | 配电室列表 / 新建 |
| PUT/DELETE | `/api/rooms/:id` | 更新 / 删除（级联其下柜与记录） |
| GET/POST | `/api/cabinets` | 配电柜列表（`?room_id=`）/ 新建 |
| PUT/DELETE | `/api/cabinets/:id` | 更新 / 删除（记录保留、柜置空） |
| GET | `/api/equipment` | 列表 `?room_id=&cabinet_id=&keyword=&page=&page_size=` |
| GET/POST | `/api/equipment[/:id]` | 单条 / 新建 |
| PUT/DELETE | `/api/equipment/:id` | 更新 / 删除 |
| POST | `/api/annex/upload` | multipart 上传图片（字段 `file`，≤10MB，uuid 落盘） |
| GET | `/api/annex/:id/file` | 读取图片文件 |
| POST | `/api/annex/recompute` | 全量重算引用次数 |
| POST | `/api/annex/cleanup` | 清理未引用附件 |

## 目录结构

```
├── backend/
│   ├── cmd/server/            # 入口 + go:embed 前端产物(webdist)
│   └── internal/
│       ├── config/            # 端口/数据目录配置
│       ├── model/             # 数据模型
│       ├── store/             # sqlite 访问（迁移/CRUD/引用重算）
│       ├── annex/             # 附件磁盘服务（uuid 落盘）
│       ├── handler/           # Gin 处理器与校验
│       └── router/            # 路由 + SPA 静态托管
├── frontend/src/
│   ├── api/                   # axios 接口封装
│   ├── components/            # 图片上传/缩略图/三个表单弹窗
│   ├── views/                 # RoomsView(配电室) / CabinetsView(配电柜) / ComponentsView(元器件)
│   └── utils/                 # 图片校验等
├── Makefile
└── scripts/bootstrap-go.sh
```

## 常见问题

- **端口占用**：`PORT=9090 make run` 换端口。
- **图片打不开**：检查 `./data/annex` 是否可读、文件是否被清理。
