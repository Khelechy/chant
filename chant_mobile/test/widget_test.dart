import 'package:flutter_test/flutter_test.dart';

import 'package:chant_mobile/src/chant_app.dart';

void main() {
  testWidgets('shows listener shell', (WidgetTester tester) async {
    await tester.pumpWidget(const ChantMobileApp());

    expect(find.text('Server-backed CHANT listener'), findsOneWidget);
    expect(find.text('Record WAV'), findsOneWidget);
    expect(find.text('Pick WAV'), findsOneWidget);
  });
}
