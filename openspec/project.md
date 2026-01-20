# Project Context

## Purpose
生产级的 Tetris 游戏，支持多前端（终端控制台和 Web UI），通过 WebSocket 与后端实时交互。目标是构建一个功能完整、性能优良、用户体验良好的经典方块游戏。

## Tech Stack
- **后端**: Go 1.21+
- **通信协议**: WebSocket (实时双向通信)
- **前端1**: 终端控制台 (基于 tcell 或 termbox)
- **前端2**: Web UI (HTML/CSS/JavaScript)
- **游戏引擎**: 自定义游戏逻辑引擎

## Project Conventions

### Code Style
- 使用 `gofmt` 标准格式化
- 遵循 Go 官方代码规范 (Effective Go)
- 简洁优先：避免过度设计和过早优化
- 包命名：小写、简洁、描述性
- 导出函数使用完整单词，非导出使用简写

### Architecture Patterns
- **前后端分离**: 游戏逻辑在后端，前端只负责渲染和输入
- **WebSocket 通信**: 使用 JSON 消息格式进行双向通信
- **模块化设计**: 游戏状态、逻辑、通信层分离
- **接口优先**: 定义清晰的接口便于测试和扩展

### Testing Strategy
- 核心游戏逻辑必须有单元测试覆盖
- 使用 Go 标准库 `testing` 包
- 测试覆盖率目标：核心逻辑 > 80%
- 集成测试：WebSocket 通信协议验证

### Git Workflow
- 主分支: `main`
- 功能分支: `feature/功能名称` (如 `feature/hold-piece`)
- 提交信息格式: `类型: 简短描述`
  - 类型: feat, fix, refactor, test, docs, chore
  - 示例: `feat: add hold piece functionality`
- 提交前必须通过 `go test` 和 `go vet`

## Domain Context

### Tetris 游戏规则
- 7种标准方块（I, O, T, S, Z, J, L）
- 方块旋转、移动、加速下落、硬降
- 消行得分机制
- 等级系统（速度随等级提升）
- 下一个方块预览
- 暂存方块功能（可选）
- 游戏结束条件：方块堆叠到顶部

### 通信协议
- WebSocket 连接管理
- JSON 消息格式：
  - 客户端→服务器: 输入指令（移动、旋转、暂停）
  - 服务器→客户端: 游戏状态更新（棋盘、分数、下一个方块）
- 心跳机制保持连接
- 断线重连处理

### 性能要求
- 游戏逻辑: 60 FPS
- 网络延迟: < 50ms (本地) / < 100ms (远程)
- 支持多客户端并发游戏

## Important Constraints
- 向后兼容: WebSocket 消息格式需要版本控制
- 资源限制: 单服务器支持至少 100 并发游戏
- 浏览器兼容: Web UI 支持现代浏览器（Chrome 90+, Firefox 88+, Safari 14+）

## External Dependencies
- **WebSocket 库**: `github.com/gorilla/websocket` (或类似)
- **终端 UI**: `github.com/gdamore/tcell` 或 `github.com/nsf/termbox-go`
- **无外部 API**: 游戏完全自包含，无外部服务依赖
