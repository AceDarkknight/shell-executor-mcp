# 文档目录

## 概述

本目录包含 Shell Executor MCP 系统的所有项目文档，包括需求文档、架构设计、API 接口说明和开发计划等。

## 文档列表

### 核心文档

- [`requirements.md`](requirements.md) - 需求文档
  - 项目背景和用户角色
  - 功能需求（Client 端和 Server 端）
  - 非功能需求
  - 约束条件

- [`architecture.md`](architecture.md) - 架构设计文档
  - 系统架构图（Mermaid 图）
  - 模块划分和职责
  - 核心流程设计
  - 时序图
  - 详细算法设计
  - 数据结构定义

- [`api.md`](api.md) - API 接口文档
  - MCP Tools 说明
  - 配置文件格式
  - 错误码说明

### 开发计划

- [`plan/`](plan/) - 开发计划目录
  - 包含以日期为前缀的计划文档
  - 每个计划包含详细的实现步骤和预期效果

## 文档规范

### 编写规范

- 所有文档使用中文编写
- 使用 Markdown 格式
- 包含必要的图表（如架构图、时序图）
- 代码示例使用代码块格式

### 更新规范

- 代码变更后及时更新相关文档
- 确保文档与代码保持同步
- 每次更新在文档末尾添加更新记录

## 文档阅读顺序

对于新加入项目的开发者，建议按以下顺序阅读：

1. [`requirements.md`](requirements.md) - 了解项目需求和目标
2. [`architecture.md`](architecture.md) - 理解系统架构和设计
3. [`api.md`](api.md) - 熟悉 API 接口
4. [`plan/`](plan/) - 查看开发计划和进度

## 更新记录

- 2026-01-23: 创建 README.md 文档
