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

  testWidgets('serializes semantics rect in global coordinates', (
    WidgetTester tester,
  ) async {
    // ensureInitialized() enables semantics via the plugin's own handle, which
    // debugResetForTest() disposes at the end of this test.
    FlarnessPluginBinding.ensureInitialized();

    const Key target = Key('flarness-rect-target');
    await tester.pumpWidget(
      Directionality(
        textDirection: TextDirection.ltr,
        child: Stack(
          children: <Widget>[
            Positioned(
              left: 100,
              top: 200,
              width: 60,
              height: 40,
              child: Semantics(
                key: target,
                label: 'target',
                container: true,
                child: const SizedBox.expand(),
              ),
            ),
          ],
        ),
      ),
    );
    await tester.pumpAndSettle();

    final SemanticsNode node = tester.getSemantics(find.byKey(target));
    final Rect expected = tester.getRect(find.byKey(target));
    final Rect global = FlarnessPluginBinding.debugGlobalRectForTest(node);

    expect(global.left, moreOrLessEquals(expected.left, epsilon: 0.5));
    expect(global.top, moreOrLessEquals(expected.top, epsilon: 0.5));
    expect(global.width, moreOrLessEquals(expected.width, epsilon: 0.5));
    expect(global.height, moreOrLessEquals(expected.height, epsilon: 0.5));
    expect(global.left, greaterThan(50));
    expect(global.top, greaterThan(150));

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

  test('merges traversal and inverse-hit-test child ids', () {
    expect(
      FlarnessPluginBinding.debugMergeSemanticsChildIdsForTest(
        <int>[10, 20],
        <int>[20, 30, 10, 40],
      ),
      <int>[10, 20, 30, 40],
    );
  });

  testWidgets('collects synthetic visible text labels', (
    WidgetTester tester,
  ) async {
    FlarnessPluginBinding.ensureInitialized();
    await tester.pumpWidget(
      const Directionality(
        textDirection: TextDirection.ltr,
        child: Column(
          children: <Widget>[
            Text('Members'),
            Text.rich(
              TextSpan(
                text: 'CAA',
                children: <InlineSpan>[TextSpan(text: ' Agent')],
              ),
            ),
          ],
        ),
      ),
    );
    await tester.pumpAndSettle();

    final labels = FlarnessPluginBinding.debugCollectSyntheticLabelsForTest();
    expect(labels, contains('Members'));
    expect(labels, contains('CAA Agent'));
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
