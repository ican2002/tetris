## ADDED Requirements

### Requirement: 终端渲染引擎
The system MUST provide a terminal rendering engine using tcell.

#### Scenario: 初始化终端界面
- **GIVEN** 程序启动
- **WHEN** 初始化终端 UI
- **THEN** 创建 tcell Screen
- **AND** 清空屏幕
- **AND** 设置终端标题

#### Scenario: 渲染游戏棋盘
- **GIVEN** 终端 UI 已初始化
- **WHEN** 接收游戏状态
- **THEN** 在屏幕左侧渲染 10×20 棋盘
- **AND** 使用对应颜色显示已占用单元格
- **AND** 空单元格显示为点或空格

#### Scenario: 渲染信息面板
- **GIVEN** 终端 UI 已初始化
- **WHEN** 接收游戏状态
- **THEN** 在屏幕右侧渲染信息面板
- **AND** 显示分数、等级、消除行数
- **AND** 显示下一个方块预览
- **AND** 显示游戏状态

#### Scenario: 刷新屏幕
- **GIVEN** 游戏状态更新
- **WHEN** 调用刷新方法
- **THEN** 重新绘制所有组件
- **AND** 调用 Screen.Show() 更新显示

#### Scenario: 清理资源
- **GIVEN** 终端 UI 运行中
- **WHEN** 程序退出
- **THEN** 调用 Screen.Fini()
- **AND** 恢复终端原始状态

### Requirement: 方块颜色渲染
The system MUST render Tetris pieces with correct colors.

#### Scenario: I 方块颜色
- **GIVEN** 渲染 I 方块
- **WHEN** 方块颜色为青色 (#00FFFF)
- **THEN** 使用 tcell.ColorCyan 渲染

#### Scenario: O 方块颜色
- **GIVEN** 渲染 O 方块
- **WHEN** 方块颜色为黄色 (#FFFF00)
- **THEN** 使用 tcell.ColorYellow 渲染

#### Scenario: 其他方块颜色
- **GIVEN** 渲染其他方块
- **WHEN** 方块有特定颜色
- **THEN** 将十六进制颜色映射到最接近的 tcell 颜色

### Requirement: 键盘输入处理
The system MUST handle keyboard input for game controls.

#### Scenario: 左移
- **GIVEN** 游戏进行中
- **WHEN** 用户按左箭头键
- **THEN** 发送 move_left 命令到服务器

#### Scenario: 右移
- **GIVEN** 游戏进行中
- **WHEN** 用户按右箭头键
- **THEN** 发送 move_right 命令到服务器

#### Scenario: 软降
- **GIVEN** 游戏进行中
- **WHEN** 用户按下箭头键
- **THEN** 发送 move_down 命令到服务器

#### Scenario: 旋转
- **GIVEN** 游戏进行中
- **WHEN** 用户按上箭头键
- **THEN** 发送 rotate 命令到服务器

#### Scenario: 硬降
- **GIVEN** 游戏进行中
- **WHEN** 用户按空格键
- **THEN** 发送 hard_drop 命令到服务器

#### Scenario: 暂停/继续
- **GIVEN** 游戏进行中或已暂停
- **WHEN** 用户按 P 键
- **THEN** 发送 pause 或 resume 命令

#### Scenario: 退出游戏
- **GIVEN** 游戏运行中
- **WHEN** 用户按 Q 或 ESC 键
- **THEN** 关闭 WebSocket 连接
- **AND** 退出程序

### Requirement: WebSocket 客户端
The system MUST connect to the WebSocket server and handle real-time updates.

#### Scenario: 连接到服务器
- **GIVEN** 程序启动
- **WHEN** 初始化 WebSocket 客户端
- **THEN** 连接到 ws://localhost:8080/ws
- **AND** 建立双向通信

#### Scenario: 接收状态更新
- **GIVEN** WebSocket 已连接
- **WHEN** 服务器发送状态消息
- **THEN** 解析 JSON 消息
- **AND** 更新本地游戏状态
- **AND** 触发屏幕重绘

#### Scenario: 发送控制命令
- **GIVEN** 用户按下控制键
- **WHEN** 生成命令消息
- **THEN** 序列化为 JSON
- **AND** 发送到 WebSocket 服务器

#### Scenario: 处理连接错误
- **GIVEN** WebSocket 连接失败
- **WHEN** 发生网络错误
- **THEN** 显示错误消息
- **AND** 尝试重新连接（最多 3 次）
- **AND** 失败后退出程序

#### Scenario: 处理游戏结束
- **GIVEN** 游戏进行中
- **WHEN** 接收 game_over 消息
- **THEN** 显示最终分数
- **AND** 等待用户按键退出

### Requirement: 响应式布局
The system MUST adapt to different terminal sizes.

#### Scenario: 最小尺寸检查
- **GIVEN** 程序启动
- **WHEN** 检测终端尺寸
- **THEN** 如果宽度 < 80 或高度 < 24
- **AND** 显示错误消息并退出

#### Scenario: 窗口大小变化
- **GIVEN** 游戏运行中
- **WHEN** 终端窗口大小改变
- **THEN** 重新计算布局
- **AND** 重新绘制界面

### Requirement: 自动重连机制
The system MUST automatically reconnect to the server on connection loss.

#### Scenario: 连接中断
- **GIVEN** WebSocket 已连接
- **WHEN** 连接意外断开
- **THEN** 显示断开消息
- **AND** 启动重连计时器（3 秒）

#### Scenario: 重连成功
- **GIVEN** 正在重连
- **WHEN** 重新连接成功
- **THEN** 显示重连成功消息
- **AND** 继续游戏

#### Scenario: 重连失败
- **GIVEN** 正在重连
- **WHEN** 重连尝试失败
- **THEN** 显示重连失败消息
- **AND** 继续尝试（最多 5 次）

### Requirement: UI 组件
The system MUST provide reusable UI components.

#### Scenario: 边框组件
- **GIVEN** 需要绘制容器
- **WHEN** 使用边框组件
- **THEN** 绘制 Unicode 边框字符
- **AND** 支持标题和样式

#### Scenario: 文本标签
- **GIVEN** 需要显示文本
- **WHEN** 使用标签组件
- **THEN** 在指定位置渲染文本
- **AND** 支持颜色和样式

#### Scenario: 方块预览框
- **GIVEN** 需要显示下一个方块
- **WHEN** 使用方块预览组件
- **THEN** 在小网格中渲染方块形状
- **AND** 使用方块颜色

### Requirement: 性能优化
The system MUST render efficiently to ensure smooth gameplay.

#### Scenario: 局部重绘
- **GIVEN** 只有部分区域变化
- **WHEN** 更新显示
- **THEN** 只重绘变化的区域
- **AND** 不是整个屏幕

#### Scenario: 帧率控制
- **GIVEN** 游戏运行中
- **WHEN** 接收状态更新
- **THEN** 限制最大帧率为 60 FPS
- **AND** 避免过度绘制

### Requirement: 用户体验
The system MUST provide good user experience.

#### Scenario: 启动画面
- **GIVEN** 程序启动
- **WHEN** 初始化完成
- **THEN** 显示欢迎信息
- **AND** 显示控制说明
- **AND** 等待服务器连接

#### Scenario: 加载指示器
- **GIVEN** 正在连接服务器
- **WHEN** 连接未建立
- **THEN** 显示动画加载指示器
- **AND** 显示连接状态

#### Scenario: 错误提示
- **GIVEN** 发生错误
- **WHEN** 需要提示用户
- **THEN** 在底部状态栏显示错误
- **AND** 使用红色高亮显示

#### Scenario: 游戏结束画面
- **GIVEN** 游戏结束
- **WHEN** 接收 game_over 消息
- **THEN** 显示游戏结束标题
- **AND** 显示最终分数
- **AND** 显示按键退出提示
