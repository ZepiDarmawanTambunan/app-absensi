import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../../../../core/face/face_capture_result.dart';
import '../../../../core/models/attendance_log.dart';
import '../../../../core/widgets/app_button.dart';
import '../../../../core/widgets/app_text_field.dart';
import '../bloc/attendance_bloc.dart';
import 'face_capture_page.dart';
import 'widgets/face_camera_view.dart';

/// Check-in / check-out screen.
///
/// TODO: replace the employee_no / employee_id text fields with an employee
/// picker backed by `GET /api/v1/employees`.
class AttendancePage extends StatefulWidget {
  final FacePageDependencies faceDeps;

  const AttendancePage({super.key, required this.faceDeps});

  @override
  State<AttendancePage> createState() => _AttendancePageState();
}

class _AttendancePageState extends State<AttendancePage> {
  final _employeeNoController = TextEditingController();
  final _employeeIdController = TextEditingController();
  int _pendingCount = 0;

  @override
  void initState() {
    super.initState();
    context.read<AttendanceBloc>().add(const AttendancePendingCountRequested());
  }

  @override
  void dispose() {
    _employeeNoController.dispose();
    _employeeIdController.dispose();
    super.dispose();
  }

  String get _employeeNo => _employeeNoController.text.trim();

  int? get _employeeId => int.tryParse(_employeeIdController.text.trim());

  bool _requireEmployeeNo() {
    if (_employeeNo.isEmpty) {
      ScaffoldMessenger.of(context)
        ..hideCurrentSnackBar()
        ..showSnackBar(
            const SnackBar(content: Text('Isi nomor pegawai dulu.')));
      return false;
    }
    return true;
  }

  Future<void> _checkInWithFace() async {
    if (!_requireEmployeeNo()) return;
    final employeeId = _employeeId;
    if (employeeId == null) {
      ScaffoldMessenger.of(context)
        ..hideCurrentSnackBar()
        ..showSnackBar(const SnackBar(
            content: Text('Isi ID pegawai (angka) untuk verifikasi wajah.')));
      return;
    }
    final results = await Navigator.of(context).push<List<FaceCaptureResult>>(
      MaterialPageRoute(
        builder: (_) => FaceCapturePage(
          deps: widget.faceDeps,
          framesToCapture: 1,
        ),
      ),
    );
    if (!mounted || results == null || results.isEmpty) return;
    final frame = results.first;
    context.read<AttendanceBloc>().add(AttendanceCheckInFaceSubmitted(
          employeeNo: _employeeNo,
          employeeId: employeeId,
          embedding: frame.embedding,
          livenessScore: frame.livenessScore,
          challengeId: frame.challengeId,
        ));
  }

  void _showMessage(String message) {
    ScaffoldMessenger.of(context)
      ..hideCurrentSnackBar()
      ..showSnackBar(SnackBar(content: Text(message)));
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Absensi'),
        actions: [
          IconButton(
            tooltip: 'Sinkronkan antrean offline',
            icon: const Icon(Icons.sync),
            onPressed: () => context
                .read<AttendanceBloc>()
                .add(const AttendancePendingSyncRequested()),
          ),
        ],
      ),
      body: BlocConsumer<AttendanceBloc, AttendanceState>(
        listener: (context, state) {
          if (state is AttendanceRecorded) {
            final log = state.log;
            final kind =
                log.type == AttendanceType.checkIn ? 'Check-in' : 'Check-out';
            _showMessage('$kind tercatat (${log.method}).');
            context
                .read<AttendanceBloc>()
                .add(const AttendancePendingCountRequested());
          } else if (state is AttendanceQueued) {
            _showMessage(
                'Offline — tersimpan di antrean, akan disinkronkan otomatis.');
            context
                .read<AttendanceBloc>()
                .add(const AttendancePendingCountRequested());
          } else if (state is AttendanceFaceRejected) {
            final r = state.result;
            _showMessage(r.rejectReason == 'liveness_too_low'
                ? 'Wajah tidak lolos liveness check.'
                : 'Wajah tidak cocok (distance ${r.distance.toStringAsFixed(3)}).');
          } else if (state is AttendanceSynced) {
            _showMessage(state.count == 0
                ? 'Tidak ada antrean yang perlu disinkronkan.'
                : '${state.count} antrean tersinkronkan.');
            context
                .read<AttendanceBloc>()
                .add(const AttendancePendingCountRequested());
          } else if (state is AttendancePendingCount) {
            setState(() => _pendingCount = state.count);
          } else if (state is AttendanceFailure) {
            _showMessage(state.message);
          }
        },
        builder: (context, state) {
          final busy = state is AttendanceInProgress;
          return SafeArea(
            child: SingleChildScrollView(
              padding: const EdgeInsets.all(24),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  if (_pendingCount > 0)
                    Card(
                      color: Theme.of(context).colorScheme.tertiaryContainer,
                      child: Padding(
                        padding: const EdgeInsets.all(12),
                        child: Text(
                          '$_pendingCount absensi menunggu sinkron (offline).',
                          textAlign: TextAlign.center,
                        ),
                      ),
                    ),
                  if (_pendingCount > 0) const SizedBox(height: 12),
                  AppTextField(
                    controller: _employeeNoController,
                    label: 'Nomor pegawai (cth: EMP001)',
                    textInputAction: TextInputAction.next,
                  ),
                  const SizedBox(height: 12),
                  AppTextField(
                    controller: _employeeIdController,
                    label: 'ID pegawai (angka, untuk wajah)',
                    keyboardType: TextInputType.number,
                    textInputAction: TextInputAction.done,
                  ),
                  const SizedBox(height: 20),
                  AppButton(
                    label: 'Check-in dengan wajah',
                    loading: false,
                    onPressed: busy ? null : _checkInWithFace,
                  ),
                  const SizedBox(height: 12),
                  AppButton(
                    label: 'Check-in manual',
                    loading: busy,
                    onPressed: busy
                        ? null
                        : () {
                            if (!_requireEmployeeNo()) return;
                            context.read<AttendanceBloc>().add(
                                AttendanceCheckInManualSubmitted(_employeeNo));
                          },
                  ),
                  const SizedBox(height: 12),
                  OutlinedButton(
                    onPressed: busy
                        ? null
                        : () {
                            if (!_requireEmployeeNo()) return;
                            context.read<AttendanceBloc>().add(
                                AttendanceCheckOutSubmitted(_employeeNo));
                          },
                    child: const Text('Check-out'),
                  ),
                ],
              ),
            ),
          );
        },
      ),
    );
  }
}
