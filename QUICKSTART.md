# 🚀 快速启动指南

## 1. 启动后端（AI交易系统 + API服务器）

```bash
# 在项目根目录
go build -o nofx-auto
./nofx-auto
```

**输出信息：**
```
╔════════════════════════════════════════════════════════════╗
║       🤖 AI驱动的币安合约自动交易系统                        ║
╚════════════════════════════════════════════════════════════╝

🤖 使用DeepSeek AI
✓ AI驱动自动交易器初始化成功

🌐 API服务器启动在 http://localhost:8080
📊 API文档:
  • GET  /api/status          - 系统状态
  • GET  /api/account         - 账户信息
  • GET  /api/positions       - 持仓列表
  • GET  /api/decisions       - 决策日志（最近30条）
  • GET  /api/decisions/latest - 最新决策（最近5条）
  • GET  /api/statistics      - 统计信息
  • GET  /health              - 健康检查

🚀 AI驱动自动交易系统启动
💰 初始余额: 958.77 USDT
⚙️  扫描间隔: 3m0s
```

## 2. 启动前端（Web Dashboard）

**新建终端窗口，执行：**

```bash
# 进入web目录
cd web

# 安装依赖（首次运行）
npm install

# 启动开发服务器
npm run dev
```

**输出信息：**
```
  VITE v6.0.7  ready in 500 ms

  ➜  Local:   http://localhost:3000/
  ➜  Network: use --host to expose
  ➜  press h + enter to show help
```

## 3. 访问Dashboard

打开浏览器访问：**http://localhost:3000**

## 📊 Dashboard功能

### 实时监控面板
- ✅ **系统状态卡** - 运行状态、AI提供商、决策周期数
- ✅ **账户概览** - 净值、可用余额、总盈亏、持仓数
- ✅ **统计信息** - 总周期、成功/失败、开仓/平仓统计
- ✅ **持仓表格** - 所有持仓的详细信息（实时更新）
- ✅ **决策日志** - 最近5条AI决策记录

### AI思维链分析
每个决策卡片包含：
- 💭 **完整的思维链** - 点击展开查看AI的4步分析过程
  - 第一步：现有持仓分析
  - 第二步：账户风险评估
  - 第三步：新机会评估
  - 第四步：最终决策总结
- 📋 **决策动作** - 开仓/平仓的详细信息（币种、杠杆、价格）
- 💰 **账户快照** - 决策时的账户状态
- 📝 **执行日志** - 每个操作的执行结果
- ❌ **错误信息** - 失败原因（如果有）

### 自动刷新
- 系统状态、账户信息、持仓列表：**每5秒刷新**
- 决策日志、统计信息：**每10秒刷新**

## 🛑 停止系统

在两个终端窗口中分别按 `Ctrl+C` 停止后端和前端服务。

## 📝 测试API

可以使用curl测试后端API：

```bash
# 获取系统状态
curl http://localhost:8080/api/status

# 获取账户信息
curl http://localhost:8080/api/account

# 获取持仓列表
curl http://localhost:8080/api/positions

# 获取最新决策
curl http://localhost:8080/api/decisions/latest
```

## ⚠️ 注意事项

1. **确保配置正确** - 检查 `main.go` 中的API密钥和配置
2. **网络连接** - 需要访问币安API和AI API
3. **端口占用** - 确保8080和3000端口未被占用
4. **Node版本** - 前端需要Node.js >= 18.0.0

## 🎯 下一步

- 观察AI的决策过程（展开思维链分析）
- 监控账户盈亏变化
- 查看持仓的实时价格和盈亏
- 分析AI的决策逻辑和成功率

**祝交易顺利！** 🚀
