import 'package:equatable/equatable.dart';

/// Employee record from `GET /api/v1/employees`.
class Employee extends Equatable {
  final int id;
  final String employeeNo;
  final String name;
  final String department;
  final bool isActive;

  const Employee({
    required this.id,
    required this.employeeNo,
    required this.name,
    required this.department,
    required this.isActive,
  });

  factory Employee.fromJson(Map<String, dynamic> json) => Employee(
        id: (json['id'] as num).toInt(),
        employeeNo: json['employee_no'] as String,
        name: json['name'] as String,
        department: json['department']?.toString() ?? '',
        isActive: json['is_active'] as bool? ?? true,
      );

  @override
  List<Object?> get props => [id, employeeNo, name, department, isActive];
}
