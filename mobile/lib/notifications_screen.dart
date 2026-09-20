import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';

import 'main.dart';

class NotificationsScreen extends StatefulWidget {
  const NotificationsScreen({super.key});

  @override
  State<NotificationsScreen> createState() => _NotificationsScreenState();
}

class _NotificationsScreenState extends State<NotificationsScreen> {
  bool isLoading = true;
  String? errorMessage;

  List<dynamic> notifications = [];

  @override
  void initState() {
    super.initState();
    loadNotifications();
  }

  Future<String?> getToken() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getString('access_token');
  }

  Future<void> loadNotifications() async {
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
        Uri.parse('$baseUrl/api/v1/notifications'),
        headers: {'Authorization': 'Bearer $token'},
      );

      if (response.statusCode == 200) {
        setState(() {
          notifications = jsonDecode(response.body);
          isLoading = false;
        });
        return;
      }

      setState(() {
        errorMessage = 'Не удалось загрузить уведомления';
        isLoading = false;
      });
    } catch (_) {
      setState(() {
        errorMessage = 'Не удалось подключиться к серверу';
        isLoading = false;
      });
    }
  }

  Future<void> markAsRead(int id) async {
    final token = await getToken();

    if (token == null || token.isEmpty) {
      return;
    }

    try {
      final response = await http.patch(
        Uri.parse('$baseUrl/api/v1/notifications/$id/read'),
        headers: {'Authorization': 'Bearer $token'},
      );

      if (response.statusCode == 200) {
        await loadNotifications();
      }
    } catch (_) {}
  }

  IconData notificationIcon(String type) {
    switch (type) {
      case 'appointment_created':
        return Icons.calendar_month;

      case 'appointment_confirmed':
        return Icons.check_circle;

      case 'appointment_cancelled':
        return Icons.cancel;

      default:
        return Icons.notifications;
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Уведомления'),
        actions: [
          IconButton(
            onPressed: loadNotifications,
            icon: const Icon(Icons.refresh),
          ),
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

    if (notifications.isEmpty) {
      return const Center(child: Text('Уведомлений пока нет'));
    }

    return RefreshIndicator(
      onRefresh: loadNotifications,
      child: ListView.builder(
        padding: const EdgeInsets.all(16),
        itemCount: notifications.length,
        itemBuilder: (context, index) {
          final notification = notifications[index];

          final id = notification['id'] as int;
          final isRead = notification['is_read'] ?? false;

          return Card(
            margin: const EdgeInsets.only(bottom: 12),
            child: ListTile(
              leading: CircleAvatar(
                child: Icon(notificationIcon(notification['type'] ?? '')),
              ),
              title: Text(
                notification['title'] ?? '',
                style: TextStyle(
                  fontWeight: isRead ? FontWeight.normal : FontWeight.bold,
                ),
              ),
              subtitle: Text(notification['message'] ?? ''),
              trailing: isRead ? null : const Icon(Icons.circle, size: 10),
              onTap: isRead
                  ? null
                  : () {
                      markAsRead(id);
                    },
            ),
          );
        },
      ),
    );
  }
}
