import 'dart:convert';

import 'package:equatable/equatable.dart';

import '../storage/key_value_store.dart';

/// A check-in/out that could not reach the server and waits for retry.
class PendingAction extends Equatable {
  final String id;
  final String kind; // 'check_in' | 'check_out'
  final String employeeNo;
  final DateTime recordedAt;
  final int attempts;

  const PendingAction({
    required this.id,
    required this.kind,
    required this.employeeNo,
    required this.recordedAt,
    this.attempts = 0,
  });

  Map<String, dynamic> toJson() => {
        'id': id,
        'kind': kind,
        'employee_no': employeeNo,
        'recorded_at': recordedAt.toIso8601String(),
        'attempts': attempts,
      };

  factory PendingAction.fromJson(Map<String, dynamic> json) => PendingAction(
        id: json['id'] as String,
        kind: json['kind'] as String,
        employeeNo: json['employee_no'] as String,
        recordedAt: DateTime.parse(json['recorded_at'] as String),
        attempts: (json['attempts'] as num?)?.toInt() ?? 0,
      );

  PendingAction withAttempts(int value) => PendingAction(
        id: id,
        kind: kind,
        employeeNo: employeeNo,
        recordedAt: recordedAt,
        attempts: value,
      );

  @override
  List<Object?> get props => [id, kind, employeeNo, recordedAt, attempts];
}

/// Durable FIFO queue of attendance actions recorded while offline.
///
/// Backed by plain key-value storage as a JSON list — small, simple, and
/// testable without plugins (see [InMemoryStore]).
class PendingQueue {
  static const storageKey = 'pending_attendance_queue_v1';

  final KeyValueStore _store;

  PendingQueue(this._store);

  Future<List<PendingAction>> all() async {
    final raw = await _store.read(storageKey);
    if (raw == null || raw.isEmpty) return [];
    final list = jsonDecode(raw) as List<dynamic>;
    return list
        .map((e) => PendingAction.fromJson(e as Map<String, dynamic>))
        .toList(growable: false);
  }

  Future<void> enqueue(PendingAction action) async {
    final items = await all();
    await _persist([...items, action]);
  }

  Future<void> remove(String id) async {
    final items = await all();
    await _persist(items.where((e) => e.id != id).toList());
  }

  Future<void> clear() => _store.remove(storageKey);

  Future<void> _persist(List<PendingAction> items) async {
    await _store.write(
        storageKey, jsonEncode(items.map((e) => e.toJson()).toList()));
  }
}
