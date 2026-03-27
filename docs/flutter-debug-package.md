# Flutter Debug Package

`flarness` 的 Go 侧交互命令现在依赖 Flutter App 在 debug 模式注册
`ext.flarness.*` service extensions。

仓库内置的 package 在：

- [packages/flarness_debug](/Users/tcn/WorkSpace/Programming/Tools/flarness/packages/flarness_debug)

## 接入方式

在 Flutter App 的 `pubspec.yaml` 里加 path 依赖：

```yaml
dependencies:
  flarness_debug:
    path: /absolute/path/to/flarness/packages/flarness_debug
```

在 `main.dart` 里初始化：

```dart
import 'package:flarness_debug/flarness_debug.dart';
import 'package:flutter/material.dart';

void main() {
  FlarnessDebugBinding.ensureInitialized();
  runApp(const MyApp());
}
```

## 当前已注册的扩展

- `ext.flarness.ping`
- `ext.flarness.tapAt`
- `ext.flarness.type`
- `ext.flarness.swipe`
- `ext.flarness.semanticsAction`

## 命令映射

- `flarness tap`
  Go 侧先解析 semantics tree，再调用 `ext.flarness.tapAt`
- `flarness type`
  直接调用 `ext.flarness.type`
- `flarness swipe`
  调用 `ext.flarness.swipe`
- `flarness scroll`
  Go 侧通过 semantics action 映射到 `ext.flarness.semanticsAction`
- `flarness longpress`
  Go 侧通过 semantics action 映射到 `ext.flarness.semanticsAction`

## 已知限制

- `type` 依赖当前有焦点的 `EditableText`
- `scroll/longpress` 依赖 Flutter semantics tree 可用
- 如设备默认未生成 semantics，仍需开启辅助功能或确保 debug 语义树可用
