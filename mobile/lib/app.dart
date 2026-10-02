import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import 'core/face/face_detection_service.dart';
import 'core/face/face_embedding_service.dart';
import 'core/face/liveness_service.dart';
import 'core/storage/key_value_store.dart';
import 'features/attendance/bloc/attendance_bloc.dart';
import 'features/attendance/domain/attendance_service.dart';
import 'features/auth/bloc/auth_bloc.dart';
import 'features/auth/domain/auth_service.dart';
import 'features/auth/presentation/login_page.dart';
import 'features/enrollment/bloc/enrollment_bloc.dart';
import 'features/enrollment/domain/enrollment_service.dart';
import 'features/face/data/face_repository.dart';
import 'features/history/bloc/history_bloc.dart';
import 'features/history/domain/history_service.dart';
import 'home_page.dart';

/// Manual dependency graph — Widget → Bloc → Service → Repository → ApiClient.
///
/// Everything is constructed once here and handed down; no service locator,
/// no globals.
class AppDependencies {
  final AuthService authService;
  final AttendanceService attendanceService;
  final EnrollmentService enrollmentService;
  final HistoryService historyService;

  final FaceDetectionService detectionService;
  final FaceEmbeddingService embeddingService;
  final LivenessService livenessService;
  final FaceRepository faceRepository;
  final KeyValueStore kvStore;

  const AppDependencies({
    required this.authService,
    required this.attendanceService,
    required this.enrollmentService,
    required this.historyService,
    required this.detectionService,
    required this.embeddingService,
    required this.livenessService,
    required this.faceRepository,
    required this.kvStore,
  });
}

class App extends StatelessWidget {
  final AppDependencies deps;

  const App({super.key, required this.deps});

  @override
  Widget build(BuildContext context) {
    return MultiBlocProvider(
      providers: [
        BlocProvider(
          create: (_) => AuthBloc(deps.authService)
            ..add(const AuthAppStarted()),
        ),
        BlocProvider(create: (_) => AttendanceBloc(deps.attendanceService)),
        BlocProvider(create: (_) => EnrollmentBloc(deps.enrollmentService)),
        BlocProvider(create: (_) => HistoryBloc(deps.historyService)),
      ],
      child: MaterialApp(
        title: 'app-absensi',
        debugShowCheckedModeBanner: false,
        theme: ThemeData(
          colorScheme: ColorScheme.fromSeed(seedColor: Colors.indigo),
          useMaterial3: true,
        ),
        home: _AuthGate(deps: deps),
      ),
    );
  }
}

/// Shows [HomePage] when authenticated, [LoginPage] otherwise.
class _AuthGate extends StatelessWidget {
  final AppDependencies deps;

  const _AuthGate({required this.deps});

  @override
  Widget build(BuildContext context) {
    return BlocBuilder<AuthBloc, AuthState>(
      builder: (context, state) {
        if (state is AuthAuthenticated) return HomePage(deps: deps);
        if (state is AuthLoading || state is AuthInitial) {
          return const Scaffold(
            body: Center(child: CircularProgressIndicator()),
          );
        }
        // AuthUnauthenticated / AuthFailure → login (failure shows a snackbar
        // on the login page itself).
        return const LoginPage();
      },
    );
  }
}
