import 'dart:async';
import 'dart:convert';
import 'dart:developer' as developer;
import 'dart:ui' as ui;

import 'package:flutter/foundation.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter/scheduler.dart';
import 'package:flutter/widgets.dart';

/// Registers the `ext.flarness.*` VM service extensions used by the Go daemon.
class FlarnessPluginBinding {
  FlarnessPluginBinding._();

  static bool _initialized = false;
  static SemanticsHandle? _semanticsHandle;
  static int _pointerCounter = 1000;

  /// Enables Flarness service extensions in debug builds.
  static void ensureInitialized() {
    if (_initialized || !kDebugMode) {
      return;
    }

    WidgetsFlutterBinding.ensureInitialized();
    _initialized = true;

    _semanticsHandle ??= SemanticsBinding.instance.ensureSemantics();

    developer.registerExtension('ext.flarness.ping', _handlePing);
    developer.registerExtension(
      'ext.flarness.dumpSemantics',
      _handleDumpSemantics,
    );
    developer.registerExtension('ext.flarness.tapAt', _handleTapAt);
    developer.registerExtension('ext.flarness.type', _handleType);
    developer.registerExtension('ext.flarness.swipe', _handleSwipe);
    developer.registerExtension(
      'ext.flarness.semanticsAction',
      _handleSemanticsAction,
    );
  }

  static Future<developer.ServiceExtensionResponse> _handlePing(
    String method,
    Map<String, String> parameters,
  ) async {
    return _ok(<String, Object?>{
      'status': 'ok',
      'message': 'pong',
      'method': method,
    });
  }

  static Future<developer.ServiceExtensionResponse> _handleTapAt(
    String method,
    Map<String, String> parameters,
  ) async {
    try {
      final x = _parseDouble(parameters['x']);
      final y = _parseDouble(parameters['y']);
      if (x == null || y == null) {
        return _error('x and y are required');
      }

      await _dispatchTap(ui.Offset(x, y));
      final EditableTextState? editable = _findEditableTextAt(ui.Offset(x, y));
      if (editable != null) {
        editable.widget.focusNode.requestFocus();
        editable.requestKeyboard();
        await SchedulerBinding.instance.endOfFrame;
      }
      return _ok(<String, Object?>{
        'status': 'ok',
        'action': 'tapAt',
        'x': x,
        'y': y,
        'focused_editable': editable != null,
      });
    } catch (error, stackTrace) {
      return _error('$error', stackTrace: stackTrace);
    }
  }

  static Future<developer.ServiceExtensionResponse> _handleDumpSemantics(
    String method,
    Map<String, String> parameters,
  ) async {
    try {
      final owner = _semanticsOwner();
      if (owner == null) {
        return _error(
          'semantics owner unavailable; ensure flarness_plugin is initialized in debug mode',
        );
      }

      final root = owner.rootSemanticsNode;
      if (root == null) {
        return _error(
          'semantics tree unavailable; ensure flarness_plugin is initialized after WidgetsFlutterBinding',
        );
      }

      final payload = <String, Object?>{
        'status': 'ok',
        'nodes': [_serializeSemanticsNode(root)],
      };
      return _ok(payload);
    } catch (error, stackTrace) {
      return _error('$error', stackTrace: stackTrace);
    }
  }

  static Future<developer.ServiceExtensionResponse> _handleType(
    String method,
    Map<String, String> parameters,
  ) async {
    try {
      final text = parameters['text'] ?? '';
      final clear = _parseBool(parameters['clear']);
      final append = _parseBool(parameters['append']);

      final editable = _findFocusedEditableText();
      if (editable == null) {
        return _error(
          'no focused EditableText found; tap a text field before calling type',
        );
      }

      final controller = editable.widget.controller;
      final currentText = controller.text;
      final nextText = clear ? '' : (append ? '$currentText$text' : text);
      final nextValue = TextEditingValue(
        text: nextText,
        selection: TextSelection.collapsed(offset: nextText.length),
        composing: TextRange.empty,
      );
      editable.widget.focusNode.requestFocus();
      editable.requestKeyboard();
      await SchedulerBinding.instance.endOfFrame;
      editable.userUpdateTextEditingValue(
        nextValue,
        SelectionChangedCause.keyboard,
      );
      await _waitForTextCommit(controller, nextText);

      return _ok(<String, Object?>{
        'status': 'ok',
        'action': 'type',
        'text': nextText,
        'current_text': currentText,
        'observed_text': controller.text,
        'focused': editable.widget.focusNode.hasFocus,
        'clear': clear,
        'append': append,
      });
    } catch (error, stackTrace) {
      return _error('$error', stackTrace: stackTrace);
    }
  }

  static Future<developer.ServiceExtensionResponse> _handleSwipe(
    String method,
    Map<String, String> parameters,
  ) async {
    try {
      final x1 = _parseDouble(parameters['x1']);
      final y1 = _parseDouble(parameters['y1']);
      final x2 = _parseDouble(parameters['x2']);
      final y2 = _parseDouble(parameters['y2']);
      final durationMs = _parseInt(parameters['duration']) ?? 300;

      if (x1 == null || y1 == null || x2 == null || y2 == null) {
        return _error('x1, y1, x2, y2, and duration are required');
      }

      await _dispatchSwipe(
        from: ui.Offset(x1, y1),
        to: ui.Offset(x2, y2),
        duration: Duration(milliseconds: durationMs),
      );
      return _ok(<String, Object?>{
        'status': 'ok',
        'action': 'swipe',
        'from': <String, double>{'x': x1, 'y': y1},
        'to': <String, double>{'x': x2, 'y': y2},
        'duration_ms': durationMs,
      });
    } catch (error, stackTrace) {
      return _error('$error', stackTrace: stackTrace);
    }
  }

  static Future<developer.ServiceExtensionResponse> _handleSemanticsAction(
    String method,
    Map<String, String> parameters,
  ) async {
    try {
      final nodeId = _parseInt(parameters['nodeId']);
      final actionName = parameters['action'];
      final owner = _semanticsOwner();
      if (nodeId == null || actionName == null || actionName.isEmpty) {
        return _error('nodeId and action are required');
      }
      if (owner == null) {
        return _error(
          'semantics owner unavailable; ensure semantics is enabled',
        );
      }

      final action = _parseSemanticsAction(actionName);
      if (action == null) {
        return _error('unsupported semantics action: $actionName');
      }

      final args = _parseActionArgs(parameters['args']);
      owner.performAction(nodeId, action, args);
      await SchedulerBinding.instance.endOfFrame;

      return _ok(<String, Object?>{
        'status': 'ok',
        'action': 'semanticsAction',
        'node_id': nodeId,
        'semantics_action': actionName,
      });
    } catch (error, stackTrace) {
      return _error('$error', stackTrace: stackTrace);
    }
  }

  static Future<void> _dispatchTap(ui.Offset position) async {
    final int pointer = _nextPointer();
    GestureBinding.instance.handlePointerEvent(
      PointerAddedEvent(
        pointer: pointer,
        position: position,
        kind: ui.PointerDeviceKind.mouse,
      ),
    );
    GestureBinding.instance.handlePointerEvent(
      PointerHoverEvent(
        pointer: pointer,
        position: position,
        kind: ui.PointerDeviceKind.mouse,
      ),
    );
    GestureBinding.instance.handlePointerEvent(
      PointerDownEvent(
        pointer: pointer,
        position: position,
        buttons: kPrimaryMouseButton,
        kind: ui.PointerDeviceKind.mouse,
      ),
    );
    await Future<void>.delayed(const Duration(milliseconds: 40));
    GestureBinding.instance.handlePointerEvent(
      PointerUpEvent(
        pointer: pointer,
        position: position,
        kind: ui.PointerDeviceKind.mouse,
      ),
    );
    GestureBinding.instance.handlePointerEvent(
      PointerRemovedEvent(
        pointer: pointer,
        position: position,
        kind: ui.PointerDeviceKind.mouse,
      ),
    );
    await SchedulerBinding.instance.endOfFrame;
  }

  static Future<void> _dispatchSwipe({
    required ui.Offset from,
    required ui.Offset to,
    required Duration duration,
  }) async {
    final int pointer = _nextPointer();
    const steps = 12;
    GestureBinding.instance.handlePointerEvent(
      PointerAddedEvent(
        pointer: pointer,
        position: from,
        kind: ui.PointerDeviceKind.mouse,
      ),
    );
    GestureBinding.instance.handlePointerEvent(
      PointerDownEvent(
        pointer: pointer,
        position: from,
        buttons: kPrimaryMouseButton,
        kind: ui.PointerDeviceKind.mouse,
      ),
    );

    final stepDelay = Duration(
      milliseconds: (duration.inMilliseconds / steps).round().clamp(1, 1000),
    );
    for (var i = 1; i <= steps; i++) {
      final t = i / steps;
      final point = ui.Offset.lerp(from, to, t) ?? to;
      await Future<void>.delayed(stepDelay);
      GestureBinding.instance.handlePointerEvent(
        PointerMoveEvent(
          pointer: pointer,
          position: point,
          buttons: kPrimaryMouseButton,
          kind: ui.PointerDeviceKind.mouse,
        ),
      );
    }

    GestureBinding.instance.handlePointerEvent(
      PointerUpEvent(
        pointer: pointer,
        position: to,
        kind: ui.PointerDeviceKind.mouse,
      ),
    );
    GestureBinding.instance.handlePointerEvent(
      PointerRemovedEvent(
        pointer: pointer,
        position: to,
        kind: ui.PointerDeviceKind.mouse,
      ),
    );
    await SchedulerBinding.instance.endOfFrame;
  }

  static EditableTextState? _findFocusedEditableText() {
    final Element? context =
        FocusManager.instance.primaryFocus?.context as Element?;
    EditableTextState? focused;

    EditableTextState? inspect(Element element) {
      if (element is StatefulElement && element.state is EditableTextState) {
        return element.state as EditableTextState;
      }
      return null;
    }

    EditableTextState? walkDescendants(Element root) {
      EditableTextState? match;
      void visit(Element element) {
        if (match != null) {
          return;
        }
        final candidate = inspect(element);
        if (candidate != null) {
          match = candidate;
          return;
        }
        element.visitChildElements(visit);
      }

      visit(root);
      return match;
    }

    if (context != null) {
      focused = inspect(context);
      if (focused != null) {
        return focused;
      }

      focused = walkDescendants(context);
      if (focused != null) {
        return focused;
      }

      context.visitAncestorElements((Element element) {
        focused = inspect(element);
        return focused == null;
      });
      if (focused != null) {
        return focused;
      }
    }

    final Element? root = WidgetsBinding.instance.rootElement;
    if (root == null) {
      return null;
    }

    void visit(Element element) {
      if (focused != null) {
        return;
      }
      final candidate = inspect(element);
      if (candidate != null && candidate.widget.focusNode.hasFocus) {
        focused = candidate;
        return;
      }
      element.visitChildElements(visit);
    }

    visit(root);
    return focused;
  }

  static EditableTextState? _findEditableTextAt(ui.Offset globalPosition) {
    EditableTextState? match;
    double bestDistance = double.infinity;
    final Element? root = WidgetsBinding.instance.rootElement;
    if (root == null) {
      return null;
    }

    void walk(Element element) {
      if (match != null) {
        return;
      }

      if (element is StatefulElement && element.state is EditableTextState) {
        final EditableTextState state = element.state as EditableTextState;
        final RenderObject? renderObject = state.context.findRenderObject();
        if (renderObject is RenderBox &&
            renderObject.attached &&
            renderObject.hasSize) {
          final ui.Offset origin = renderObject.localToGlobal(ui.Offset.zero);
          final ui.Rect rect = origin & renderObject.size;
          if (rect.contains(globalPosition)) {
            match = state;
            return;
          }
          double dx = 0;
          if (globalPosition.dx < rect.left) {
            dx = rect.left - globalPosition.dx;
          } else if (globalPosition.dx > rect.right) {
            dx = globalPosition.dx - rect.right;
          }

          double dy = 0;
          if (globalPosition.dy < rect.top) {
            dy = rect.top - globalPosition.dy;
          } else if (globalPosition.dy > rect.bottom) {
            dy = globalPosition.dy - rect.bottom;
          }
          final double distance = ui.Offset(dx, dy).distance;
          if (distance < bestDistance && distance <= 32) {
            bestDistance = distance;
            match = state;
          }
        }
      }

      element.visitChildElements(walk);
    }

    walk(root);
    return match;
  }

  static Object? _parseActionArgs(String? raw) {
    if (raw == null || raw.isEmpty) {
      return null;
    }
    try {
      return jsonDecode(raw);
    } catch (_) {
      return raw;
    }
  }

  static SemanticsAction? _parseSemanticsAction(String name) {
    switch (name) {
      case 'tap':
        return SemanticsAction.tap;
      case 'longPress':
        return SemanticsAction.longPress;
      case 'scrollLeft':
        return SemanticsAction.scrollLeft;
      case 'scrollRight':
        return SemanticsAction.scrollRight;
      case 'scrollUp':
        return SemanticsAction.scrollUp;
      case 'scrollDown':
        return SemanticsAction.scrollDown;
      case 'showOnScreen':
        return SemanticsAction.showOnScreen;
      case 'increase':
        return SemanticsAction.increase;
      case 'decrease':
        return SemanticsAction.decrease;
      default:
        return null;
    }
  }

  static bool _parseBool(String? raw) {
    if (raw == null) {
      return false;
    }
    return raw == 'true' || raw == '1';
  }

  static int? _parseInt(String? raw) {
    if (raw == null || raw.isEmpty) {
      return null;
    }
    return int.tryParse(raw);
  }

  static double? _parseDouble(String? raw) {
    if (raw == null || raw.isEmpty) {
      return null;
    }
    return double.tryParse(raw);
  }

  static SemanticsOwner? _semanticsOwner() {
    // `pipelineOwner` is deprecated in newer Flutter builds but remains the
    // most widely available path across current stable releases.
    // ignore: deprecated_member_use
    return RendererBinding.instance.pipelineOwner.semanticsOwner ??
        RendererBinding.instance.rootPipelineOwner.semanticsOwner;
  }

  static Future<developer.ServiceExtensionResponse> _ok(
    Map<String, Object?> payload,
  ) async {
    return developer.ServiceExtensionResponse.result(jsonEncode(payload));
  }

  static Future<developer.ServiceExtensionResponse> _error(
    String message, {
    StackTrace? stackTrace,
  }) async {
    return developer.ServiceExtensionResponse.result(
      jsonEncode(<String, Object?>{
        'status': 'error',
        'error': message,
        if (stackTrace != null) 'stack': '$stackTrace',
      }),
    );
  }

  static int _nextPointer() {
    _pointerCounter += 1;
    return _pointerCounter;
  }

  static Map<String, Object?> _serializeSemanticsNode(SemanticsNode node) {
    final rect = node.rect;
    final data = node.getSemanticsData();
    return <String, Object?>{
      'id': node.id,
      'label': node.label,
      'value': node.value,
      'hint': node.hint,
      'rect': <String, Object?>{
        'left': rect.left,
        'top': rect.top,
        'width': rect.width,
        'height': rect.height,
      },
      'actions': _serializeActions(data.actions),
      'flags': data.flagsCollection.toStrings(),
      'children': node
          .debugListChildrenInOrder(DebugSemanticsDumpOrder.traversalOrder)
          .map(_serializeSemanticsNode)
          .toList(growable: false),
    };
  }

  static List<String> _serializeActions(int actions) {
    final values = <String>[];
    for (final action in SemanticsAction.values) {
      if (actions & action.index != 0) {
        values.add(_describeEnum(action));
      }
    }
    return values;
  }

  static String _describeEnum(Object value) {
    final raw = value.toString();
    final dot = raw.indexOf('.');
    if (dot >= 0 && dot + 1 < raw.length) {
      return raw.substring(dot + 1);
    }
    return raw;
  }

  static Future<void> _waitForTextCommit(
    TextEditingController controller,
    String expectedText,
  ) async {
    final DateTime deadline = DateTime.now().add(
      const Duration(milliseconds: 800),
    );

    while (DateTime.now().isBefore(deadline)) {
      await SchedulerBinding.instance.endOfFrame;
      if (controller.text == expectedText) {
        await Future<void>.delayed(const Duration(milliseconds: 120));
        await SchedulerBinding.instance.endOfFrame;
        return;
      }
      await Future<void>.delayed(const Duration(milliseconds: 16));
    }

    await SchedulerBinding.instance.endOfFrame;
  }
}
