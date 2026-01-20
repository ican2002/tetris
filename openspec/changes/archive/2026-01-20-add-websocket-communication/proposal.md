# Change: 添加 WebSocket 通信层

## Why
当前游戏引擎已完成，但缺少前后端通信机制。需要实现 WebSocket 通信层，使终端和 Web 前端能够与游戏引擎实时交互。

## What Changes
- 实现 WebSocket 服务器，支持多客户端连接
- 定义 JSON 消息协议（客户端→服务器、服务器→客户端）
- 实现连接管理（连接、断开、会话管理）
- 实现消息路由和分发
- 实现心跳机制保持连接活跃
- 实现断线重连支持
- 集成现有游戏引擎

## Impact
- **Affected specs**: 无（这是新功能）
- **Affected code**:
  - 新增 `pkg/server/` 包（WebSocket 服务器）
  - 新增 `pkg/protocol/` 包（消息协议定义）
  - 更新 `pkg/game/` 包（添加状态变化通知）
- **Dependencies**:
  - `github.com/gorilla/websocket` - WebSocket 实现
- **Testing**: 需要集成测试验证通信协议

## Technical Notes
- 使用 JSON 作为消息格式
- 支持同时多个游戏客户端
- 每个连接维护独立的游戏会话
- 心跳间隔：30 秒
- 消息类型：控制、状态、错误、心跳
