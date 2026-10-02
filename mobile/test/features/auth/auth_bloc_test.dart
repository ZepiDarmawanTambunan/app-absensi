import 'package:bloc_test/bloc_test.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import 'package:app_absensi/core/models/user.dart';
import 'package:app_absensi/core/network/api_exception.dart';
import 'package:app_absensi/features/auth/bloc/auth_bloc.dart';

import '../../helpers/mocks.dart';

void main() {
  const user = AppUser(id: 1, name: 'Admin', email: 'admin@example.com');

  group('AuthBloc', () {
    late MockAuthService service;

    setUp(() {
      service = MockAuthService();
    });

    blocTest<AuthBloc, AuthState>(
      'restores an existing session on app start',
      setUp: () =>
          when(() => service.restoreSession()).thenAnswer((_) async => user),
      build: () => AuthBloc(service),
      act: (bloc) => bloc.add(const AuthAppStarted()),
      expect: () => [const AuthLoading(), const AuthAuthenticated(user)],
    );

    blocTest<AuthBloc, AuthState>(
      'shows login when no session exists',
      setUp: () =>
          when(() => service.restoreSession()).thenAnswer((_) async => null),
      build: () => AuthBloc(service),
      act: (bloc) => bloc.add(const AuthAppStarted()),
      expect: () => [const AuthLoading(), const AuthUnauthenticated()],
    );

    blocTest<AuthBloc, AuthState>(
      'authenticates on valid credentials',
      setUp: () => when(() => service.login(
            email: 'admin@example.com',
            password: 'secret123',
          )).thenAnswer((_) async => user),
      build: () => AuthBloc(service),
      act: (bloc) => bloc.add(const AuthLoginSubmitted(
          email: 'admin@example.com', password: 'secret123')),
      expect: () => [const AuthLoading(), const AuthAuthenticated(user)],
    );

    blocTest<AuthBloc, AuthState>(
      'surfaces the backend message on login failure',
      setUp: () => when(() => service.login(
            email: 'a@b.c',
            password: 'salah',
          )).thenThrow(const ApiException(
          code: 'unauthorized', message: 'Email atau kata sandi salah.')),
      build: () => AuthBloc(service),
      act: (bloc) =>
          bloc.add(const AuthLoginSubmitted(email: 'a@b.c', password: 'salah')),
      expect: () => [
        const AuthLoading(),
        const AuthFailure('Email atau kata sandi salah.')
      ],
    );

    blocTest<AuthBloc, AuthState>(
      'logs out cleanly',
      setUp: () => when(() => service.logout()).thenAnswer((_) async {}),
      build: () => AuthBloc(service),
      act: (bloc) => bloc.add(const AuthLoggedOut()),
      expect: () => [const AuthUnauthenticated()],
      verify: (_) => verify(() => service.logout()).called(1),
    );
  });
}
