import 'package:demo_app/main.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  testWidgets('creates an issue from the composer and selects it', (
    WidgetTester tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(1440, 1800));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    await tester.pumpWidget(const MyApp());

    await tester.tap(
      find.byKey(const ValueKey<String>('open-create-issue-button')),
    );
    await tester.pumpAndSettle();

    await tester.enterText(
      find.descendant(
        of: find.byKey(const ValueKey<String>('issue-title-input')),
        matching: find.byType(EditableText),
      ),
      '下班',
    );
    await tester.ensureVisible(
      find.byKey(const ValueKey<String>('create-issue-button')),
    );
    await tester.tap(find.byKey(const ValueKey<String>('create-issue-button')));
    await tester.pumpAndSettle();

    expect(find.text('Created FL-104 下班'), findsOneWidget);
    expect(find.text('FL-104'), findsWidgets);
    expect(find.text('下班'), findsWidgets);
    expect(find.text('Start 下班'), findsOneWidget);
  });

  testWidgets('moves the selected issue through the workflow', (
    WidgetTester tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(1440, 1800));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    await tester.pumpWidget(const MyApp());

    await tester.tap(find.byKey(const ValueKey<String>('issue-row-FL-102')));
    await tester.pumpAndSettle();
    expect(find.text('Tighten daemon restart recovery'), findsWidgets);

    await tester.ensureVisible(find.text('Start Tighten daemon restart recovery'));
    await tester.tap(find.text('Start Tighten daemon restart recovery'));
    await tester.pumpAndSettle();
    expect(
      find.text('Moved FL-102 Tighten daemon restart recovery to In Progress'),
      findsOneWidget,
    );
    expect(
      find.text('Complete Tighten daemon restart recovery'),
      findsOneWidget,
    );

    await tester.ensureVisible(find.text('Complete Tighten daemon restart recovery'));
    await tester.tap(find.text('Complete Tighten daemon restart recovery'));
    await tester.pumpAndSettle();
    expect(
      find.text('Completed FL-102 Tighten daemon restart recovery'),
      findsOneWidget,
    );
    expect(find.text('Reopen Tighten daemon restart recovery'), findsOneWidget);
  });

  testWidgets('search and filter narrow the issue list', (
    WidgetTester tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(1440, 1800));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    await tester.pumpWidget(const MyApp());

    await tester.enterText(
      find.byKey(const ValueKey<String>('search-input')),
      'semantics',
    );
    await tester.pumpAndSettle();

    expect(find.text('Ship semantics tree snapshots'), findsWidgets);
    expect(find.text('Tighten daemon restart recovery'), findsNothing);

    await tester.tap(find.byKey(const ValueKey<String>('filter-done')));
    await tester.pumpAndSettle();
    expect(find.text('Ship semantics tree snapshots'), findsWidgets);

    await tester.enterText(
      find.byKey(const ValueKey<String>('search-input')),
      '',
    );
    await tester.pumpAndSettle();
    expect(find.text('Ship semantics tree snapshots'), findsWidgets);
    expect(find.text('Polish onboarding command palette'), findsNothing);
  });
}
