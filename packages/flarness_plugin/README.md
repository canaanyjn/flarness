# flarness_plugin

`flarness_plugin` is the Flutter app-side integration package for Flarness.
It registers the `ext.flarness.*` VM service extensions that the Flarness CLI
uses for UI automation in debug builds.

## What it provides

- `ext.flarness.ping`
- `ext.flarness.dumpSemantics`
- `ext.flarness.tapAt`
- `ext.flarness.type`
- `ext.flarness.swipe`
- `ext.flarness.semanticsAction`

It also keeps Flutter semantics enabled in debug mode so `flarness semantics`
and the `flarness interact ...` subcommands can resolve nodes from the
semantics tree.

## Install

For local development, add a path dependency from your Flutter app:

```yaml
dependencies:
  flarness_plugin:
    path: ../path/to/flarness/packages/flarness_plugin
```

For external use, prefer a git dependency pinned to a tag:

```yaml
dependencies:
  flarness_plugin:
    git:
      url: https://github.com/canaanyjn/flarness.git
      path: packages/flarness_plugin
      ref: v0.1.0
```

## Usage

Initialize it before `runApp`:

```dart
import 'package:flarness_plugin/flarness_plugin.dart';
import 'package:flutter/material.dart';

void main() {
  FlarnessPluginBinding.ensureInitialized();
  runApp(const MyApp());
}
```

The registration is debug-only. In release/profile, `ensureInitialized()` is a
no-op.
