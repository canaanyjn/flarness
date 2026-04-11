import 'dart:convert';
import 'dart:typed_data';

import 'package:flarness_plugin/flarness_plugin.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  testWidgets('registers screenshot extension', (WidgetTester tester) async {
    FlarnessPluginBinding.ensureInitialized();

    expect(
      FlarnessPluginBinding.debugRegisteredExtensions,
      contains('ext.flarness.captureScreenshot'),
    );
    FlarnessPluginBinding.debugResetForTest();
  });

  test('rejects screenshot capture with no render views', () {
    expect(
      () => FlarnessPluginBinding.debugSelectScreenshotRenderView(
        const <RenderView>[],
      ),
      throwsA(isA<StateError>()),
    );
  });

  testWidgets('rejects ambiguous render view selection', (
    WidgetTester tester,
  ) async {
    FlarnessPluginBinding.ensureInitialized();
    await tester.pumpWidget(
      const Directionality(
        textDirection: TextDirection.ltr,
        child: SizedBox(width: 120, height: 80),
      ),
    );
    await tester.pumpAndSettle();

    final RenderView view = RendererBinding.instance.renderViews.single;
    expect(
      () => FlarnessPluginBinding.debugSelectScreenshotRenderView(
        <RenderView>[view, view],
      ),
      throwsA(isA<StateError>()),
    );
    FlarnessPluginBinding.debugResetForTest();
  });

  testWidgets('encodes screenshot payload as png', (WidgetTester tester) async {
    FlarnessPluginBinding.ensureInitialized();

    final Map<String, Object?> payload =
        FlarnessPluginBinding.debugBuildScreenshotPayload(
      Uint8List.fromList(_pngSignature),
      width: 3,
      height: 2,
      pixelRatio: 2.0,
    );

    expect(payload['status'], 'ok');
    expect(payload['format'], 'png');
    expect(payload['width'], 3);
    expect(payload['height'], 2);
    expect(payload['pixel_ratio'], 2.0);

    final String imageBase64 = payload['image_base64']! as String;
    final List<int> bytes = base64Decode(imageBase64);
    expect(bytes.take(8).toList(), <int>[
      0x89,
      0x50,
      0x4E,
      0x47,
      0x0D,
      0x0A,
      0x1A,
      0x0A,
    ]);
    FlarnessPluginBinding.debugResetForTest();
  });
}

const List<int> _pngSignature = <int>[
  0x89,
  0x50,
  0x4E,
  0x47,
  0x0D,
  0x0A,
  0x1A,
  0x0A,
];
