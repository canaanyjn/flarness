# flarness_debug

`flarness_debug` is a debug-only Flutter package that registers `ext.flarness.*`
VM service extensions for the Flarness CLI.

## What it provides

- `ext.flarness.ping`
- `ext.flarness.tapAt`
- `ext.flarness.type`
- `ext.flarness.swipe`
- `ext.flarness.semanticsAction`

It also keeps Flutter semantics enabled in debug mode so `flarness semantics`,
`tap`, `wait`, `scroll`, and `longpress` can resolve nodes from the semantics
tree.

## Install

Add a path dependency from your Flutter app:

```yaml
dependencies:
  flarness_debug:
    path: ../path/to/flarness/packages/flarness_debug
```

## Usage

Initialize it before `runApp`:

```dart
import 'package:flarness_debug/flarness_debug.dart';
import 'package:flutter/material.dart';

void main() {
  FlarnessDebugBinding.ensureInitialized();
  runApp(const MyApp());
}
```

The registration is debug-only. In release/profile, `ensureInitialized()` is a
no-op.
