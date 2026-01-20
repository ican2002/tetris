# Change: 添加 Tetris 基础游戏引擎

## Why
当前项目没有游戏逻辑，需要实现完整的 Tetris 核心引擎作为整个系统的基础。这是构建双前端游戏的前提条件。

## What Changes
- 实现标准的 7 种 Tetris 方块（I, O, T, S, Z, J, L）
- 实现 10x20 的标准游戏棋盘
- 实现方块操作：左移、右移、旋转、软降、硬降
- 实现碰撞检测系统
- 实现行消除和得分计算
- 实现游戏状态管理（进行中、暂停、游戏结束）
- 实现下一个方块预览功能
- 实现等级和速度系统

## Impact
- **Affected specs**: 无（这是初始功能）
- **Affected code**:
  - 新增 `pkg/game/` 包
  - 新增 `pkg/piece/` 包（方块定义）
  - 新增 `pkg/board/` 包（棋盘逻辑）
- **Dependencies**: 暂无外部依赖（纯 Go 实现）
- **Testing**: 需要全面的单元测试覆盖核心逻辑
