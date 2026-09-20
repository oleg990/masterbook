import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile/main.dart';

void main() {
  testWidgets(
    'Экран входа MasterBook',
    (WidgetTester tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: LoginScreen(),
        ),
      );

      expect(find.text('MasterBook'), findsOneWidget);
      expect(find.text('Войти'), findsOneWidget);
      expect(find.text('Регистрация'), findsOneWidget);
    },
  );
}