# Flutter Plugin Package

`flarness` 的 Go 侧交互命令依赖 Flutter App 在 debug 模式注册
`ext.flarness.*` service extensions。Flutter 侧建议通过
`flarness_plugin` 接入。

仓库内置的 package 在：

- [packages/flarness_plugin](/Users/tcn/WorkSpace/Programming/Tools/flarness/packages/flarness_plugin)

## 接入方式

本地联调时，在 Flutter App 的 `pubspec.yaml` 里加 path 依赖：

```yaml
dependencies:
  flarness_plugin:
    path: /absolute/path/to/flarness/packages/flarness_plugin
```

外部项目更适合直接走 git 依赖，并固定到 release tag：

```yaml
dependencies:
  flarness_plugin:
    git:
      url: https://github.com/canaanyjn/flarness.git
      path: packages/flarness_plugin
      ref: v0.1.0
```

在 `main.dart` 里初始化：

```dart
import 'package:flarness_plugin/flarness_plugin.dart';
import 'package:flutter/material.dart';

void main() {
  FlarnessPluginBinding.ensureInitialized();
  runApp(const MyApp());
}
```

这个包的定位是“Flutter 侧接入插件”，但运行行为仍然是 debug-only。
在 release/profile 下，初始化是 no-op，不会把 Flarness 调试扩展带进正式构建。

## 当前已注册的扩展

- `ext.flarness.ping`
- `ext.flarness.dumpSemantics`
- `ext.flarness.tapAt`
- `ext.flarness.type`
- `ext.flarness.swipe`
- `ext.flarness.semanticsAction`

## 命令映射

- `flarness interact tap`
  Go 侧先解析 semantics tree，再调用 `ext.flarness.tapAt`
- `flarness interact type`
  直接调用 `ext.flarness.type`
- `flarness interact swipe`
  调用 `ext.flarness.swipe`
- `flarness interact scroll`
  Go 侧通过 semantics action 映射到 `ext.flarness.semanticsAction`
- `flarness interact longpress`
  Go 侧通过 semantics action 映射到 `ext.flarness.semanticsAction`

## 已知限制

- `type` 依赖当前有焦点的 `EditableText`
- `scroll/longpress` 依赖 Flutter semantics tree 可用
- 如设备默认未生成 semantics，仍需开启辅助功能或确保 debug 语义树可用
