# Change: 添加终端用户界面

## Why
当前游戏引擎和 WebSocket 服务器已完成，但缺少用户可交互的前端。需要实现终端 UI，使用户可以在命令行中体验完整的 Tetris 游戏。

## What Changes
- 实现基于 tcell 的终端渲染引擎
- 实现游戏棋盘的彩色显示
- 实现方块预览和状态信息面板
- 实现键盘输入处理（方向键、空格等）
- 实现 WebSocket 客户端连接
- 实现实时状态更新
- 实现游戏循环和动画

## Impact
- **Affected specs**: 无（这是新功能）
- **Affected code**:
  - 新增 `pkg/tui/` 包（终端 UI 组件）
  - 新增 `pkg/wsclient/` 包（WebSocket 客户端）
  - 新增 `cmd/tetris/` 目录（终端游戏程序）
- **Dependencies**:
  - `github.com/gdamore/tcell/v2` - 终端 UI 库
  - 现有 WebSocket 服务器
- **Testing**: 需要集成测试验证 UI 交互

## Technical Notes
- 使用 tcell v2 进行终端渲染
- 支持 Unicode 字符和 256 色
- 支持 80×24 最小终端尺寸
- 键盘控制：方向键移动、空格硬降、P 暂停、Q 退出
- 自动重连 WebSocket 服务器
- 响应式布局适应终端大小
