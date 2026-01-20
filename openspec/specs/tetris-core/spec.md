# tetris-core Specification

## Purpose
TBD - created by archiving change add-tetris-game-engine. Update Purpose after archive.
## Requirements
### Requirement: 方块定义
The system MUST support 7 standard Tetris pieces, each with a fixed color and shape.
系统必须支持 7 种标准 Tetris 方块，每种方块有固定的颜色和形状。

#### Scenario: I 方块（青色）
- **GIVEN** 系统初始化
- **WHEN** 创建 I 方块
- **THEN** 方块形状为 4 个单元格的竖条（`[[1,1,1,1]]`）
- **AND** 颜色为青色（Cyan，#00FFFF）

#### Scenario: O 方块（黄色）
- **GIVEN** 系统初始化
- **WHEN** 创建 O 方块
- **THEN** 方块形状为 2x2 的正方形（`[[1,1],[1,1]]`）
- **AND** 颜色为黄色（Yellow，#FFFF00）

#### Scenario: T 方块（紫色）
- **GIVEN** 系统初始化
- **WHEN** 创建 T 方块
- **THEN** 方块形状为 T 形（`[[0,1,0],[1,1,1]]`）
- **AND** 颜色为紫色（Purple，#800080）

#### Scenario: S 方块（绿色）
- **GIVEN** 系统初始化
- **WHEN** 创建 S 方块
- **THEN** 方块形状为 S 形（`[[0,1,1],[1,1,0]]`）
- **AND** 颜色为绿色（Green，#00FF00）

#### Scenario: Z 方块（红色）
- **GIVEN** 系统初始化
- **WHEN** 创建 Z 方块
- **THEN** 方块形状为 Z 形（`[[1,1,0],[0,1,1]]`）
- **AND** 颜色为红色（Red，#FF0000）

#### Scenario: J 方块（蓝色）
- **GIVEN** 系统初始化
- **WHEN** 创建 J 方块
- **THEN** 方块形状为 J 形（`[[1,0,0],[1,1,1]]`）
- **AND** 颜色为蓝色（Blue，#0000FF）

#### Scenario: L 方块（橙色）
- **GIVEN** 系统初始化
- **WHEN** 创建 L 方块
- **THEN** 方块形状为 L 形（`[[0,0,1],[1,1,1]]`）
- **AND** 颜色为橙色（Orange，#FFA500）

### Requirement: 游戏棋盘
The system MUST maintain a standard 10x20 game board and track placed pieces.
系统必须维护一个 10x20 的标准游戏棋盘，能够追踪已放置的方块。

#### Scenario: 初始化空棋盘
- **GIVEN** 系统启动
- **WHEN** 创建新游戏
- **THEN** 棋盘宽度为 10 列
- **AND** 棋盘高度为 20 行
- **AND** 所有单元格为空

#### Scenario: 方块放置到棋盘
- **GIVEN** 游戏进行中
- **WHEN** 当前方块锁定到棋盘
- **THEN** 方块占据的单元格被标记为已占用
- **AND** 单元格记录方块颜色

#### Scenario: 访问棋盘单元格
- **GIVEN** 棋盘已初始化
- **WHEN** 查询坐标 (x, y) 的单元格
- **THEN** 如果 x 在 [0,9] 且 y 在 [0,19] 范围内，返回该单元格状态
- **AND** 如果超出范围，返回错误

### Requirement: 方块生成
The system MUST use the 7-bag randomization algorithm to generate pieces, ensuring even distribution.
系统必须使用 7-bag 随机算法生成方块，确保方块分布均匀。

#### Scenario: 7-bag 随机生成
- **GIVEN** 游戏开始
- **WHEN** 需要生成新方块
- **THEN** 创建包含所有 7 种方块的袋子
- **AND** 随机打乱袋子中的方块
- **AND** 按顺序返回袋子中的方块
- **AND** 袋子为空时重新填充

#### Scenario: 下一个方块预览
- **GIVEN** 游戏进行中
- **WHEN** 查询下一个方块
- **THEN** 返回袋子中的下一个方块（不移除）
- **AND** 如果袋子为空，先填充再返回

### Requirement: 方块移动
The system MUST support piece movement operations including left, right, and down.
系统必须支持方块的移动操作，包括左移、右移和下落。

#### Scenario: 方块左移
- **GIVEN** 当前活动方块存在
- **WHEN** 执行左移操作
- **AND** 左侧没有障碍物
- **THEN** 方块 x 坐标减 1
- **AND** 方块位置更新成功

#### Scenario: 方块左移被阻挡
- **GIVEN** 当前活动方块存在
- **WHEN** 执行左移操作
- **AND** 左侧有边界或已占用单元格
- **THEN** 方块位置不变
- **AND** 操作失败但不报错

#### Scenario: 方块右移
- **GIVEN** 当前活动方块存在
- **WHEN** 执行右移操作
- **AND** 右侧没有障碍物
- **THEN** 方块 x 坐标加 1

#### Scenario: 方块软降
- **GIVEN** 当前活动方块存在
- **WHEN** 执行软降操作
- **AND** 下方没有障碍物
- **THEN** 方块 y 坐标加 1

#### Scenario: 方块硬降
- **GIVEN** 当前活动方块存在
- **WHEN** 执行硬降操作
- **THEN** 方块立即下落到最低可能位置
- **AND** 方块被锁定到棋盘
- **AND** 触发行消除检查

### Requirement: 方块旋转
The system MUST support clockwise 90-degree piece rotation and handle wall kicks.
系统必须支持方块顺时针旋转 90 度，并处理墙踢（wall kick）。

#### Scenario: 标准旋转
- **GIVEN** 当前活动方块存在
- **WHEN** 执行顺时针旋转
- **AND** 旋转后位置没有障碍物
- **THEN** 方块顺时针旋转 90 度
- **AND** 方块位置更新

#### Scenario: 墙踢 - I 方块
- **GIVEN** I 方块在墙边
- **WHEN** 执行旋转导致碰撞
- **THEN** 尝试向左或向右平移 1-2 个单位
- **AND** 如果找到有效位置，执行旋转和平移
- **AND** 否则旋转失败

#### Scenario: 墙踢 - 其他方块
- **GIVEN** 非 I 方块在墙边
- **WHEN** 执行旋转导致碰撞
- **THEN** 尝试向左或向右平移 1 个单位
- **AND** 如果找到有效位置，执行旋转和平移
- **AND** 否则旋转失败

#### Scenario: O 方块不旋转
- **GIVEN** O 方块存在
- **WHEN** 执行旋转操作
- **THEN** 方块不改变（O 方块旋转后形状相同）

### Requirement: 碰撞检测
The system MUST accurately detect collisions between pieces and boundaries or placed pieces.
系统必须准确检测方块与边界、已放置方块的碰撞。

#### Scenario: 边界碰撞检测
- **GIVEN** 方块在棋盘中
- **WHEN** 方块的任何单元格超出棋盘边界
- **THEN** 系统检测到碰撞
- **AND** 返回碰撞状态

#### Scenario: 方块碰撞检测
- **GIVEN** 棋盘上有已放置的方块
- **WHEN** 当前方块与已占用单元格重叠
- **THEN** 系统检测到碰撞
- **AND** 返回碰撞状态

#### Scenario: 无碰撞情况
- **GIVEN** 方块在棋盘中央
- **WHEN** 方块所有单元格在边界内
- **AND** 所有目标单元格为空
- **THEN** 系统检测无碰撞
- **AND** 允许操作继续

### Requirement: 行消除
The system MUST detect and clear complete rows, updating the board and score.
系统必须检测并消除完整的行，并更新棋盘和分数。

#### Scenario: 单行消除
- **GIVEN** 棋盘有一行完全被占用
- **WHEN** 锁定方块后检查行
- **THEN** 识别完整行
- **AND** 移除该行
- **AND** 上方所有行下移 1 格
- **AND** 顶部生成新的空行

#### Scenario: 多行消除（Tetris）
- **GIVEN** 棋盘有 4 行连续完全被占用
- **WHEN** 锁定方块后检查行
- **THEN** 同时消除 4 行
- **AND** 上方所有行下移 4 格

#### Scenario: 无行消除
- **GIVEN** 棋盘没有完整行
- **WHEN** 锁定方块后检查行
- **THEN** 棋盘不变
- **AND** 不更新分数

### Requirement: 得分系统
The system MUST calculate scores based on cleared rows and support hard drop bonus points.
系统必须根据消除的行数计算得分，并支持连击奖励。

#### Scenario: 单行消除得分
- **GIVEN** 玩家消除 1 行
- **WHEN** 计算得分
- **THEN** 得分为 100 × 当前等级

#### Scenario: 双行消除得分
- **GIVEN** 玩家消除 2 行
- **WHEN** 计算得分
- **THEN** 得分为 300 × 当前等级

#### Scenario: 三行消除得分
- **GIVEN** 玩家消除 3 行
- **WHEN** 计算得分
- **THEN** 得分为 500 × 当前等级

#### Scenario: Tetris（4 行）得分
- **GIVEN** 玩家消除 4 行
- **WHEN** 计算得分
- **THEN** 得分为 800 × 当前等级

#### Scenario: 硬降得分
- **GIVEN** 玩家执行硬降
- **WHEN** 方块下落 2 格
- **THEN** 额外得分 2 × 当前等级

### Requirement: 等级系统
The system MUST maintain player levels and increase levels based on cleared rows.
系统必须维护玩家等级，并根据消除行数提升等级。

#### Scenario: 初始等级
- **GIVEN** 新游戏开始
- **WHEN** 初始化游戏状态
- **THEN** 等级为 1

#### Scenario: 等级提升
- **GIVEN** 当前等级为 1
- **WHEN** 玩家累计消除 10 行
- **THEN** 等级提升至 2
- **AND** 游戏速度增加

#### Scenario: 等级速度公式
- **GIVEN** 玩家等级为 N
- **WHEN** 计算下落间隔
- **THEN** 间隔（毫秒）= max(100, 1000 - (N-1) × 100)
- **AND** 等级 1 为 1000ms，等级 10 为 100ms

### Requirement: 游戏状态管理
The system MUST maintain game states including playing, paused, and game over.
系统必须维护游戏状态，包括进行中、暂停和游戏结束。

#### Scenario: 游戏进行中
- **GIVEN** 游戏已启动
- **WHEN** 玩家可以操作方块
- **THEN** 状态为 "playing"
- **AND** 方块继续自动下落

#### Scenario: 游戏暂停
- **GIVEN** 游戏进行中
- **WHEN** 玩家请求暂停
- **THEN** 状态变为 "paused"
- **AND** 方块停止下落
- **AND** 输入被忽略（除恢复暂停）

#### Scenario: 恢复游戏
- **GIVEN** 游戏暂停中
- **WHEN** 玩家请求恢复
- **THEN** 状态变为 "playing"
- **AND** 方块继续下落

#### Scenario: 游戏结束
- **GIVEN** 新方块生成
- **WHEN** 新方块的初始位置与棋盘碰撞
- **THEN** 状态变为 "gameover"
- **AND** 方块停止下落
- **AND** 输入被忽略
- **AND** 显示最终分数

### Requirement: 游戏循环
The system MUST maintain a game loop that automatically drops pieces at fixed intervals.
系统必须维护游戏主循环，按固定间隔自动下落方块。

#### Scenario: 自动下落
- **GIVEN** 游戏状态为 "playing"
- **WHEN** 经过一个下落间隔
- **THEN** 方块自动下移 1 格
- **AND** 如果下方有障碍，方块锁定
- **AND** 生成新方块

#### Scenario: 暂停时停止循环
- **GIVEN** 游戏状态为 "paused"
- **WHEN** 经过一个下落间隔
- **THEN** 方块不移动
- **AND** 等待恢复游戏

### Requirement: 游戏数据查询
The system MUST provide interfaces to query the current game state.
系统必须提供查询当前游戏状态的接口。

#### Scenario: 查询完整游戏状态
- **GIVEN** 游戏进行中
- **WHEN** 请求游戏状态
- **THEN** 返回：棋盘状态、当前方块、下一个方块、分数、等级、消除行数、游戏状态

#### Scenario: 查询棋盘状态
- **GIVEN** 游戏进行中
- **WHEN** 请求棋盘状态
- **THEN** 返回 10x20 的二维数组，每个单元格包含颜色或空值

#### Scenario: 查询当前方块
- **GIVEN** 游戏进行中
- **WHEN** 请求当前方块
- **THEN** 返回方块类型、位置、旋转状态

