import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';

import 'main.dart';

class MasterScreen extends StatefulWidget {
  const MasterScreen({super.key});

  @override
  State<MasterScreen> createState() => _MasterScreenState();
}

class _MasterScreenState extends State<MasterScreen> {
  bool isLoading = true;
  String? errorMessage;
  List<dynamic> appointments = [];

  @override
  void initState() {
    super.initState();
    loadAppointments();
  }

  Future<String?> getToken() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getString('access_token');
  }

  Future<void> loadAppointments() async {
    setState(() {
      isLoading = true;
      errorMessage = null;
    });

    try {
      final token = await getToken();

      if (token == null || token.isEmpty) {
        setState(() {
          errorMessage = 'Необходимо войти в аккаунт';
          isLoading = false;
        });
        return;
      }

      final response = await http.get(
        Uri.parse('$baseUrl/api/v1/master/appointments'),
        headers: {'Authorization': 'Bearer $token'},
      );

      if (response.statusCode == 200) {
        setState(() {
          appointments = jsonDecode(response.body);
          isLoading = false;
        });
        return;
      }

      setState(() {
        errorMessage = 'Не удалось загрузить записи';
        isLoading = false;
      });
    } catch (_) {
      setState(() {
        errorMessage = 'Не удалось подключиться к серверу';
        isLoading = false;
      });
    }
  }

  Future<void> changeStatus(int appointmentId, String action) async {
    final token = await getToken();

    if (token == null || token.isEmpty) {
      return;
    }

    try {
      final response = await http.patch(
        Uri.parse(
          '$baseUrl/api/v1/master/appointments/'
          '$appointmentId/$action',
        ),
        headers: {'Authorization': 'Bearer $token'},
      );

      if (response.statusCode == 200) {
        if (!mounted) return;

        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(switch (action) {
              'confirm' => 'Запись подтверждена',
              'complete' => 'Запись завершена',
              _ => 'Запись отменена',
            }),
          ),
        );

        await loadAppointments();
        return;
      }

      if (!mounted) return;

      final data = jsonDecode(response.body);

      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(data['error'] ?? 'Не удалось изменить запись')),
      );
    } catch (_) {
      if (!mounted) return;

      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Ошибка соединения с сервером')),
      );
    }
  }

  Future<void> logout() async {
    final prefs = await SharedPreferences.getInstance();

    await prefs.remove('access_token');

    if (!mounted) return;

    Navigator.of(context).pushAndRemoveUntil(
      MaterialPageRoute(builder: (_) => const LoginScreen()),
      (route) => false,
    );
  }

  String statusText(String status) {
    switch (status) {
      case 'pending':
        return 'Ожидает подтверждения';
      case 'confirmed':
        return 'Подтверждена';
      case 'completed':
        return 'Завершена';
      case 'cancelled':
        return 'Отменена';
      default:
        return status;
    }
  }

  Color statusColor(String status) {
    switch (status) {
      case 'pending':
        return Colors.orange;
      case 'confirmed':
        return Colors.green;
      case 'completed':
        return Colors.blue;
      case 'cancelled':
        return Colors.red;
      default:
        return Colors.grey;
    }
  }

  String formatDateTime(String value) {
    final date = DateTime.tryParse(value);

    if (date == null) {
      return value;
    }

    return '${date.day.toString().padLeft(2, '0')}.'
        '${date.month.toString().padLeft(2, '0')}.'
        '${date.year} '
        '${date.hour.toString().padLeft(2, '0')}:'
        '${date.minute.toString().padLeft(2, '0')}';
  }

  Widget appointmentCard(dynamic appointment) {
    final appointmentId = appointment['id'] as int;
    final status = appointment['status'] ?? '';

    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              appointment['client_name'] ?? '',
              style: const TextStyle(fontSize: 20, fontWeight: FontWeight.bold),
            ),

            const SizedBox(height: 6),

            Text(appointment['client_email'] ?? ''),

            const SizedBox(height: 10),

            Text(
              appointment['service_name'] ?? '',
              style: const TextStyle(fontSize: 16),
            ),

            const SizedBox(height: 8),

            Text(formatDateTime(appointment['start_time'] ?? '')),

            const SizedBox(height: 10),

            Container(
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
              decoration: BoxDecoration(
                color: statusColor(status).withValues(alpha: 0.12),
                borderRadius: BorderRadius.circular(8),
              ),
              child: Text(
                statusText(status),
                style: TextStyle(
                  color: statusColor(status),
                  fontWeight: FontWeight.bold,
                ),
              ),
            ),

            if (status == 'pending') ...[
              const SizedBox(height: 12),
              Row(
                children: [
                  Expanded(
                    child: FilledButton(
                      onPressed: () {
                        changeStatus(appointmentId, 'confirm');
                      },
                      child: const Text('Подтвердить'),
                    ),
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: OutlinedButton(
                      onPressed: () {
                        changeStatus(appointmentId, 'cancel');
                      },
                      child: const Text('Отменить'),
                    ),
                  ),
                ],
              ),
            ],

            if (status == 'confirmed') ...[
              const SizedBox(height: 12),
              Row(
                children: [
                  Expanded(
                    child: FilledButton(
                      onPressed: () {
                        changeStatus(appointmentId, 'complete');
                      },
                      child: const Text('Завершить'),
                    ),
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: OutlinedButton(
                      onPressed: () {
                        changeStatus(appointmentId, 'cancel');
                      },
                      child: const Text('Отменить'),
                    ),
                  ),
                ],
              ),
            ],
          ],
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Кабинет мастера'),
        actions: [
          IconButton(
            onPressed: loadAppointments,
            icon: const Icon(Icons.refresh),
          ),
          IconButton(onPressed: logout, icon: const Icon(Icons.logout)),
        ],
      ),
      body: buildBody(),
    );
  }

  Widget buildBody() {
    if (isLoading) {
      return const Center(child: CircularProgressIndicator());
    }

    if (errorMessage != null) {
      return Center(child: Text(errorMessage!));
    }

    if (appointments.isEmpty) {
      return const Center(child: Text('Записей пока нет'));
    }

    return RefreshIndicator(
      onRefresh: loadAppointments,
      child: ListView(
        padding: const EdgeInsets.all(16),
        children: appointments
            .map((appointment) => appointmentCard(appointment))
            .toList(),
      ),
    );
  }
}
