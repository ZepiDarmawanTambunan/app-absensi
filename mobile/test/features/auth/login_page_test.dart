import 'package:bloc_test/bloc_test.dart';
import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import 'package:app_absensi/features/auth/bloc/auth_bloc.dart';
import 'package:app_absensi/features/auth/presentation/login_page.dart';

class MockAuthBloc extends MockBloc<AuthEvent, AuthState> implements AuthBloc {}

void main() {
  setUpAll(() {
    registerFallbackValue(const AuthAppStarted());
  });

  late MockAuthBloc bloc;

  setUp(() {
    bloc = MockAuthBloc();
    when(() => bloc.state).thenReturn(const AuthInitial());
    when(() => bloc.stream).thenAnswer((_) => const Stream<AuthState>.empty());
  });

  Future<void> pumpLogin(WidgetTester tester) {
    return tester.pumpWidget(
      MaterialApp(
        home: BlocProvider<AuthBloc>.value(
          value: bloc,
          child: const LoginPage(),
        ),
      ),
    );
  }

  testWidgets('shows validation errors and does not submit when empty',
      (tester) async {
    await pumpLogin(tester);
    await tester.tap(find.text('Masuk'));
    await tester.pump();

    expect(find.text('Email wajib diisi'), findsOneWidget);
    expect(find.text('Kata sandi wajib diisi'), findsOneWidget);
    verifyNever(() => bloc.add(any()));
  });

  testWidgets('dispatches AuthLoginSubmitted on valid input', (tester) async {
    await pumpLogin(tester);

    await tester.enterText(
        find.byType(TextFormField).first, 'admin@example.com');
    await tester.enterText(find.byType(TextFormField).at(1), 'secret123');
    await tester.tap(find.text('Masuk'));
    await tester.pump();

    verify(() => bloc.add(const AuthLoginSubmitted(
        email: 'admin@example.com', password: 'secret123'))).called(1);
  });

  testWidgets('shows a loading indicator while authenticating',
      (tester) async {
    when(() => bloc.state).thenReturn(const AuthLoading());
    await pumpLogin(tester);

    expect(find.byType(CircularProgressIndicator), findsOneWidget);
  });
}
