核心定位：让 AI Agent 像人类开发者一样驱动 Flutter 完整开发循环。
所有操作返回结构化 JSON，AI 可直接解析。

架构：Daemon + CLI 两层分离
─────────────────────────────

AI Agent ──▶ CLI 短命令 ──▶ Unix Socket IPC ──▶ Daemon(长驻) ──▶ Flutter 子进程
             (即用即走)      ~/.flarness/        (管理进程、      flutter run
                             daemon.sock         收集日志、        --machine
                                                 维护状态)

技术选型
────────

语言         Go                    单二进制分发、交叉编译、goroutine 并发
CLI 框架     cobra                 Go CLI 业界标准
IPC 通信     Unix Domain Socket    本地最快、无端口占用、安全
日志格式     JSONL                 追加写入零开销、逐行流式、grep 友好
WebSocket    gorilla/websocket     CDP / VM Service 通信
配置         gopkg.in/yaml.v3      YAML 配置管理
外部依赖     仅 3 个               极致精简

项目结构 (60 个 .go 文件, 18 个测试文件)
───────────────────────────────────────

flarness/
├── main.go                             入口, 注入版本号
├── cmd/                                CLI 命令层 (21 个命令)
│   ├── root.go                         根命令 + JSON 输出工具
│   ├── start.go / stop.go / status.go  生命周期
│   ├── reload.go / restart.go          热重载/重启
│   ├── logs.go / errors.go             日志查询
│   ├── analyze.go                      静态分析
│   ├── screenshot.go                   截图
│   ├── inspect.go                      Widget 树检查
│   ├── snapshot.go                     截图 + Widget 树组合
│   ├── tap.go / type.go                点击 / 输入文字
│   ├── scroll.go / swipe.go            滚动 / 滑动
│   ├── longpress.go / wait.go          长按 / 等待元素
│   ├── semantics.go                    语义树
│   ├── daemon.go                       隐藏: _daemon 子命令
│   ├── internal_inspect.go             隐藏: _inspect 子进程
│   └── internal_interact.go            隐藏: _interact 子进程
│
└── internal/                           内部包层 (12 个子包)
    ├── daemon/
    │   ├── daemon.go                   生命周期 (Setsid detach, PID, Socket)
    │   ├── server.go                   Unix Socket IPC 服务端
    │   └── handler.go                  命令路由 → 分发到各子系统
    │
    ├── process/
    │   └── manager.go                  flutter run --machine 子进程管理
    │                                   状态机: Idle→Starting→Running→Reloading→Stopped
    │
    ├── parser/
    │   ├── machine_parser.go           解析 --machine JSON 事件流
    │   └── stderr_parser.go            解析编译错误 (Dart/Gradle/Xcode/CocoaPods)
    │
    ├── collector/
    │   ├── collector.go                内存环形缓冲 + JSONL 文件持久化
    │   ├── query.go                    多维过滤: limit/since/level/source/grep
    │   └── rotation.go                 日志轮转 (50MB) + 过期清理
    │
    ├── cdp/
    │   └── bridge.go                   CDP WebSocket 桥接, 捕获浏览器 console
    │
    ├── nativebridge/
    │   └── logcat.go                   adb logcat 捕获 Android 日志
    │
    ├── inspector/
    │   └── inspector.go                VM Service 获取 Widget 树
    │                                   三种降级: SummaryTree → WidgetTree → debugDumpApp
    │
    ├── interaction/
    │   └── interaction.go              语义树定位 + evaluate 注入事件
    │                                   tap/type/scroll/swipe/longpress/wait
    │
    ├── snapshot/
    │   └── screenshot.go               多平台截图 (CDP/screencapture/adb/simctl)
    │
    ├── analyzer/
    │   └── analyzer.go                 flutter analyze 结构化输出
    │
    ├── ipc/
    │   └── client.go                   CLI 端 Socket 客户端
    │
    ├── config/
    │   └── config.go                   ~/.flarness/config.yaml
    │
    ├── platform/
    │   └── device.go                   设备检测, 优先级: macOS > Android > iOS > Chrome
    │
    └── model/
        ├── command.go                  IPC 命令 {Cmd, Args}
        ├── log_entry.go                日志条目 (5 级别 × 4 来源)
        └── response.go                11 种响应类型

核心数据流
──────────

flutter run --machine (子进程)
    │
    ├── stdout ──▶ MachineParser ──▶ LogCollector ──▶ 内存缓冲 + JSONL 文件
    ├── stderr ──▶ StderrParser  ──┘
    └── stdin  ◀── "r" / "R" / "q" ◀── Handler
                                           ▲
CDPBridge (Web console) ──────────────▶ LogCollector
LogcatBridge (Android logcat) ────────▶ LogCollector
                                           │
IPC Server ◀── CLI 命令                     │
    │                                       │
Handler 路由 ──▶ 子进程 _inspect/_interact   │
               (规避 WebSocket 单连接冲突)    │
                                            ▼
                                      JSON 响应返回 AI

关键技术亮点
────────────

1. 子进程隔离架构
   inspect/interaction 通过独立子进程执行,
   规避 Dart VM Service WebSocket 单连接限制。

2. 多平台截图策略
   Web→CDP, macOS→screencapture, Android→adb screencap, iOS→simctl, 逐级降级。

3. 语义树 UI 自动化
   通过 Semantics Tree 按文本/类型定位, 不依赖坐标, 抗布局变化。
   通过 Flutter Service Extension 注入事件, 零侵入。

4. 日志全覆盖
   6 大平台 (Android/iOS/macOS/Windows/Linux/Web) 均 100% 覆盖。
   Web 通过 CDP 桥接补齐浏览器 console 输出。

5. 极致精简
   仅 3 个外部依赖, 其余全部 Go 标准库。

AI Agent 闭环
─────────────

flarness start        启动 daemon + flutter run
    ↓
AI 修改代码
    ↓
flarness reload       热重载, 获取编译结果 (JSON)
    ↓
flarness snapshot     截图 + Widget 树
    ↓
flarness tap/type     操作 UI
    ↓
flarness screenshot   验证操作结果
    ↓
flarness logs         检查运行时日志
    ↓
继续修改... (循环)

落地开发计划
────────────

下面补充的是面向当前仓库状态的执行计划。
上面的内容描述目标态；本节描述“从现状走到目标态”的具体顺序。

现状判断
────────

当前仓库已具备这些基础能力：

1. Daemon + CLI + Unix Socket IPC 基本跑通
2. flutter run --machine 子进程管理已实现
3. machine/stderr 基础解析、日志采集、reload/restart 已实现
4. analyze / screenshot / inspect / snapshot 已有首版
5. Web 场景下已有 CDP console 日志桥接雏形

当前主要缺口：

1. UI 自动化能力未落地
   缺少 tap / type / scroll / swipe / longpress / wait / semantics
   AI 还不能稳定操作 Flutter UI

2. VM Service 子进程隔离未落地
   目前 inspect/snapshot 仍由 daemon 直连 VM Service
   尚未实现 _inspect / _interact 隔离模型

3. 多平台桥接未补齐
   缺少 Android logcat / iOS / macOS / 桌面平台补强
   screenshot 也未达到计划中的多策略降级能力

4. 配置和设备抽象未接入主流程
   internal/config 已存在，但 start/daemon 尚未真正消费
   platform/device 仍缺失

5. 日志生命周期能力不完整
   logs 暴露了 follow/list/clean/export 等入口
   但后端实现与 handler 路由尚不完整

6. 工程化闭环还不够稳
   命令矩阵、文档、测试反馈速度、跨平台验证都还需要补强

开发原则
────────

1. 先打通 Agent 闭环，再补平台广度
   优先保证 AI 能完成 “改代码 → reload → inspect → 操作 UI → 验证结果 → 看日志”

2. 先做稳定选择器，再做丰富手势
   先交付 semantics + tap/type/wait，再扩展 scroll/swipe/longpress

3. 先解决连接稳定性，再提高并发能力
   子进程隔离优先级高于复杂功能堆叠

4. 每一阶段都要求 JSON 契约稳定
   新命令必须先定义 response schema，再实现 handler 与测试

阶段计划
────────

Phase 0: 规划对齐与基线收敛

目标：
把“文档中的完整蓝图”和“当前代码中的真实能力”对齐，避免后续继续发散。

交付物：

1. README 命令说明更新为“已实现 / 计划中”两类
2. docs/plan.md 保留目标态，同时标出阶段路线图
3. help 命令输出中只暴露真实可用能力
4. 补一份 capability matrix
   建议文件：docs/capabilities.md

验收标准：

1. 文档不再声明不存在的命令或包
2. 新人只读 README 与 help 即可知道当前边界

Phase 1: 交付最小可用 Agent UI 闭环

目标：
让 AI 能稳定地“找元素、点击、输入、等待结果”。

范围：

1. 新增 semantics 命令
   返回语义树、节点文本、label、value、actions、bounds

2. 新增 interaction 包
   提供按文本、语义 label、widget/type 的节点查找能力

3. 新增 tap 命令
   支持按文本/label/type/节点 id 定位

4. 新增 type 命令
   支持聚焦输入框并输入文本

5. 新增 wait 命令
   支持等待元素出现、消失、可点击

6. snapshot 响应补充更多可用于 agent 推理的元信息
   如当前页面标题、路由、关键语义摘要

建议拆分：

1. internal/interaction
2. cmd/semantics.go
3. cmd/tap.go
4. cmd/type.go
5. cmd/wait.go
6. model 新增交互响应结构

验收标准：

1. 能在示例 Flutter App 上完成点击按钮
2. 能完成输入框文本输入
3. 能等待异步加载结果出现
4. 所有命令返回稳定 JSON
5. 至少有一套端到端 smoke test

Phase 2: 引入 VM Service 子进程隔离

目标：
解决 inspect / interaction 与 daemon 争用 VM Service 连接的问题，提高稳定性。

范围：

1. 实现隐藏命令 _inspect
2. 实现隐藏命令 _interact
3. daemon handler 通过子进程调用 inspect / semantics / interaction
4. 统一子进程超时、stderr 捕获、JSON 解码错误处理
5. 为 snapshot 切换到隔离执行路径

关键设计要求：

1. daemon 不长期持有 inspect/interaction 的 VM Service 连接
2. 子进程失败时返回结构化错误，不污染主守护进程状态
3. 超时可配置，并在响应中明确标识 timeout/source

验收标准：

1. 连续执行 inspect + tap + snapshot 不出现连接冲突
2. daemon 长时间运行后仍保持稳定
3. 子进程异常退出时，daemon 仍可继续服务其他命令

Phase 3: 完善手势与复杂交互

目标：
从最小交互扩展到足够覆盖常见 Flutter 页面操作。

范围：

1. 新增 scroll 命令
2. 新增 swipe 命令
3. 新增 longpress 命令
4. 支持节点 bounds 与方向性手势
5. 支持列表容器内查找与滚动直到命中

验收标准：

1. 能滚动列表直到目标元素出现
2. 能完成横向/纵向 swipe
3. 能在复杂页面中稳定找到目标节点

Phase 4: 多平台桥接与截图能力补齐

目标：
把当前偏 Web 的实现扩展为真正可跨平台使用的运行时桥接层。

范围：

1. 新增 platform/device
   统一设备检测、优先级、能力判断

2. 新增 nativebridge/logcat
   Android 原生日志接入 Collector

3. 增强 screenshot
   Web: 正确发现 CDP page target
   macOS: screencapture
   Android: adb exec-out screencap
   iOS: xcrun simctl io screenshot
   失败时统一降级路径

4. 明确各平台 capability flags
   如 supports_cdp / supports_screenshot / supports_semantics

验收标准：

1. Chrome / Android / iOS Simulator / macOS 四类设备至少覆盖三类
2. screenshot 失败时能给出明确降级路径与错误原因
3. logs 可区分 flutter/framework/app/native 来源

Phase 5: 日志生命周期与可观测性补全

目标：
让日志系统真正成为 Agent 可查询、可过滤、可维护的事实来源。

范围：

1. 补全 logs --list
2. 补全 logs --clean
3. 补全 logs --export
4. 明确 follow 是否支持 JSONL stream 或仅人类模式
5. 接入 rotation/retention 到 daemon 生命周期
6. 增加结构化错误聚合视图
   包括 compile/runtime/layout/navigation 类别

验收标准：

1. 日志轮转自动触发
2. 过期日志可清理
3. export 输出格式稳定
4. errors 命令不只是 logs 的薄封装，能输出更适合 Agent 的错误摘要

Phase 6: 配置、默认值与设备发现

目标：
让 flarness 在重复使用时减少显式参数，提高可迁移性。

范围：

1. start 读取 ~/.flarness/config.yaml
2. 支持默认 device / extra_args / log rotation config
3. 实现 device discovery
4. 增加 status 中的 capability summary
5. 统一项目级与全局级配置合并策略

验收标准：

1. 无显式 device 时可按配置或检测结果选择
2. status 输出当前配置来源与生效值
3. 配置错误会返回清晰可读的 JSON 错误

Phase 7: 质量、兼容性与发布准备

目标：
将原型推进到可持续维护的工具。

范围：

1. 扩充单元测试
   parser / collector / config / interaction selector / handler

2. 增加集成测试
   daemon 启停、reload、inspect、interaction smoke test

3. 增加示例 Flutter fixture 项目
   用于 CI 回归验证

4. 增加平台兼容性文档
5. 明确版本号、变更日志、发布脚本

验收标准：

1. go test ./... 稳定可执行并在合理时间内完成
2. 至少一条 CI 流水线覆盖核心命令 smoke test
3. 发布包在干净环境可安装并跑通 start/status/reload/snapshot

命令与能力优先级
────────────────

P0 必做：

1. semantics
2. tap
3. type
4. wait
5. _inspect
6. _interact

P1 应做：

1. scroll
2. swipe
3. platform/device
4. nativebridge/logcat
5. logs lifecycle

P2 可延后：

1. longpress
2. 更强的错误聚合
3. 更完整的 capability reporting
4. 更复杂的 route/page 摘要能力

建议实施顺序
────────────

建议按照下面的 4 个里程碑推进：

Milestone A
实现 semantics + tap + type + wait
结果：AI 首次具备最小 UI 操作能力

Milestone B
实现 _inspect + _interact 子进程隔离
结果：交互与 inspect 稳定性显著提升

Milestone C
实现 scroll/swipe + 多平台 screenshot/log bridge
结果：从 Web 优先演进到多平台可用

Milestone D
实现 logs lifecycle + config/device discovery + CI
结果：从功能原型变成可持续维护的工具

每个里程碑的完成定义
──────────────────

每个里程碑完成前，必须同时满足：

1. CLI 命令可直接使用
2. help JSON 已更新
3. model response 已定稿
4. handler 路由已接通
5. 至少包含单元测试
6. README 或 docs 已同步

近期建议
────────

如果现在立刻开始开发，我建议先做这一轮：

1. Phase 1 中的 semantics / tap / type / wait
2. 为这些命令先定义统一 selector schema
3. 同步补 model response 与 help 输出
4. 然后立刻进入 Phase 2，完成 _inspect / _interact 隔离

原因很简单：
没有交互能力，flarness 还不能形成真正的 Agent 闭环；
没有隔离能力，这套交互能力后续会在稳定性上反复返工。
