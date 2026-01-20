## ADDED Requirements

### Requirement: WebSocket 服务器
The system MUST provide a WebSocket server that accepts client connections and manages game sessions.

#### Scenario: 服务器启动
- **GIVEN** 系统初始化
- **WHEN** 启动 WebSocket 服务器
- **THEN** 服务器在指定端口监听
- **AND** 准备接受客户端连接

#### Scenario: 客户端连接
- **GIVEN** WebSocket 服务器运行中
- **WHEN** 客户端发起 WebSocket 连接
- **THEN** 连接被接受
- **AND** 创建新的游戏会话
- **AND** 发送初始游戏状态给客户端

#### Scenario: 多客户端连接
- **GIVEN** WebSocket 服务器运行中
- **WHEN** 多个客户端同时连接
- **THEN** 每个客户端获得独立的游戏会话
- **AND** 会话之间互不干扰

### Requirement: 消息协议
The system MUST define a JSON-based message protocol for client-server communication.

#### Scenario: 客户端发送控制消息
- **GIVEN** 客户端已连接
- **WHEN** 客户端发送控制消息（移动、旋转等）
- **THEN** 消息被解析为操作指令
- **AND** 游戏状态相应更新

#### Scenario: 服务器发送状态更新
- **GIVEN** 游戏状态发生变化
- **WHEN** 方块移动或锁定
- **THEN** 服务器发送完整游戏状态
- **AND** 包含棋盘、当前方块、分数等信息

#### Scenario: 消息格式验证
- **GIVEN** 客户端发送消息
- **WHEN** 消息格式不符合协议
- **THEN** 服务器返回错误消息
- **AND** 连接保持活跃

### Requirement: 控制命令处理
The system MUST process game control commands from clients.

#### Scenario: 处理移动命令
- **GIVEN** 客户端发送 `move_left` 命令
- **WHEN** 服务器接收并处理命令
- **THEN** 方块向左移动
- **AND** 发送更新后的游戏状态

#### Scenario: 处理旋转命令
- **GIVEN** 客户端发送 `rotate` 命令
- **WHEN** 服务器接收并处理命令
- **THEN** 方块旋转 90 度
- **AND** 发送更新后的游戏状态

#### Scenario: 处理硬降命令
- **GIVEN** 客户端发送 `hard_drop` 命令
- **WHEN** 服务器接收并处理命令
- **THEN** 方块立即落到底部
- **AND** 觸發行消除和得分更新

#### Scenario: 处理暂停命令
- **GIVEN** 客户端发送 `pause` 命令
- **WHEN** 服务器接收并处理命令
- **THEN** 游戏状态变为暂停
- **AND** 方块停止下落

### Requirement: 游戏状态同步
The system MUST synchronize game state with clients in real-time.

#### Scenario: 方块移动后同步
- **GIVEN** 游戏进行中
- **WHEN** 当前方块位置改变
- **THEN** 服务器发送状态更新
- **AND** 客户端收到最新的方块位置

#### Scenario: 行消除后同步
- **GIVEN** 玩家消除行
- **WHEN** 棋盘更新完成
- **THEN** 服务器发送更新后的棋盘
- **AND** 发送新的分数和等级

#### Scenario: 游戏结束通知
- **GIVEN** 新方块生成时碰撞
- **WHEN** 游戏结束条件触发
- **THEN** 服务器发送游戏结束消息
- **AND** 包含最终分数和统计

### Requirement: 心跳机制
The system MUST maintain active connections using a heartbeat mechanism.

#### Scenario: 服务器发送心跳
- **GIVEN** 客户端已连接
- **WHEN** 距离上次消息超过 30 秒
- **THEN** 服务器发送心跳消息
- **AND** 等待客户端响应

#### Scenario: 客户端响应心跳
- **GIVEN** 服务器发送心跳
- **WHEN** 客户端返回 pong
- **THEN** 连接保持活跃
- **AND** 计时器重置

#### Scenario: 心跳超时断开
- **GIVEN** 客户端无响应
- **WHEN** 超过 60 秒未收到响应
- **THEN** 服务器关闭连接
- **AND** 清理会话资源

### Requirement: 连接管理
The system MUST manage client connections and sessions.

#### Scenario: 连接断开处理
- **GIVEN** 客户端已连接
- **WHEN** 连接异常断开
- **THEN** 服务器检测到断开
- **AND** 清理游戏会话
- **AND** 释放资源

#### Scenario: 优雅关闭
- **GIVEN** 客户端发送关闭消息
- **WHEN** 服务器接收关闭请求
- **THEN** 发送关闭确认
- **AND** 清理会话
- **AND** 关闭连接

#### Scenario: 会话恢复
- **GIVEN** 客户端断开后重连
- **WHEN** 新连接建立
- **THEN** 创建新的游戏会话
- **AND** 不保留之前的状态

### Requirement: 错误处理
The system MUST handle errors gracefully and communicate them to clients.

#### Scenario: 无效命令错误
- **GIVEN** 客户端发送未知命令
- **WHEN** 服务器无法解析命令
- **THEN** 返回错误消息
- **AND** 指明无效的命令类型

#### Scenario: 游戏状态错误
- **GIVEN** 游戏已结束
- **WHEN** 客户端尝试发送控制命令
- **THEN** 返回错误消息
- **AND** 提示游戏已结束

#### Scenario: 服务器内部错误
- **GIVEN** 处理消息时发生异常
- **WHEN** 捕获到内部错误
- **THEN** 返回通用错误消息
- **AND** 记录错误日志
- **AND** 保持连接活跃

### Requirement: 并发安全
The system MUST handle concurrent operations safely.

#### Scenario: 并发消息处理
- **GIVEN** 客户端快速发送多个命令
- **WHEN** 服务器并发处理消息
- **THEN** 游戏状态保持一致
- **AND** 无竞态条件

#### Scenario: 多客户端隔离
- **GIVEN** 多个客户端同时操作
- **WHEN** 服务器处理不同会话
- **THEN** 会话之间完全隔离
- **AND** 无数据泄露

### Requirement: 消息类型定义
The system MUST support the following message types.

#### Scenario: 控制消息类型
- **GIVEN** 定义控制消息
- **THEN** 包含类型：`move_left`, `move_right`, `move_down`, `rotate`, `hard_drop`, `pause`, `resume`

#### Scenario: 状态消息类型
- **GIVEN** 定义状态消息
- **THEN** 包含：棋盘、当前方块、下一个方块、分数、等级、行数、游戏状态

#### Scenario: 系统消息类型
- **GIVEN** 定义系统消息
- **THEN** 包含：`ping`, `pong`, `error`, `game_over`
