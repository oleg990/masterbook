import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';

import 'master_screen.dart';
import 'notifications_screen.dart';

const String baseUrl = 'http://localhost:8080';

void main() {
  runApp(const MasterBookApp());
}

class MasterBookApp extends StatelessWidget {
  const MasterBookApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'MasterBook',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        useMaterial3: true,
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.pink),
      ),
      home: const StartScreen(),
    );
  }
}

// ==================================================
// START
// ==================================================

class StartScreen extends StatelessWidget {
  const StartScreen({super.key});

  Future<Widget> getStartScreen() async {
    final prefs = await SharedPreferences.getInstance();
    final token = prefs.getString('access_token');

    if (token == null || token.isEmpty) {
      return const LoginScreen();
    }

    try {
      final response = await http.get(
        Uri.parse('$baseUrl/api/v1/me'),
        headers: {'Authorization': 'Bearer $token'},
      );

      if (response.statusCode != 200) {
        await prefs.remove('access_token');
        return const LoginScreen();
      }

      final data = jsonDecode(response.body);
      final role = data['role'] ?? 'client';

      if (role == 'master') {
        return const MasterScreen();
      }

      return const ClientShell();
    } catch (_) {
      return const LoginScreen();
    }
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<Widget>(
      future: getStartScreen(),
      builder: (context, snapshot) {
        if (snapshot.connectionState != ConnectionState.done) {
          return const Scaffold(
            body: Center(child: CircularProgressIndicator()),
          );
        }

        return snapshot.data ?? const LoginScreen();
      },
    );
  }
}

// ==================================================
// LOGIN
// ==================================================

class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key});

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final emailController = TextEditingController();
  final passwordController = TextEditingController();

  bool isLoading = false;
  String? errorMessage;

  Future<String> getRole(String token) async {
    final response = await http.get(
      Uri.parse('$baseUrl/api/v1/me'),
      headers: {'Authorization': 'Bearer $token'},
    );

    if (response.statusCode != 200) {
      return 'client';
    }

    final data = jsonDecode(response.body);

    return data['role'] ?? 'client';
  }

  Future<void> login() async {
    setState(() {
      isLoading = true;
      errorMessage = null;
    });

    try {
      final response = await http.post(
        Uri.parse('$baseUrl/api/v1/auth/login'),
        headers: {'Content-Type': 'application/json'},
        body: jsonEncode({
          'email': emailController.text.trim(),
          'password': passwordController.text,
        }),
      );

      final data = jsonDecode(response.body);

      if (response.statusCode != 200) {
        setState(() {
          errorMessage = data['error'] ?? 'Ошибка входа';
        });
        return;
      }

      final token = data['access_token'];

      final prefs = await SharedPreferences.getInstance();

      await prefs.setString('access_token', token);

      final role = await getRole(token);

      if (!mounted) return;

      final nextScreen = role == 'master'
          ? const MasterScreen()
          : const ClientShell();

      Navigator.of(context).pushAndRemoveUntil(
        MaterialPageRoute(builder: (_) => nextScreen),
        (route) => false,
      );
    } catch (_) {
      setState(() {
        errorMessage = 'Не удалось подключиться к серверу';
      });
    } finally {
      if (mounted) {
        setState(() {
          isLoading = false;
        });
      }
    }
  }

  @override
  void dispose() {
    emailController.dispose();
    passwordController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(24),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 420),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  const SizedBox(height: 30),

                  const Icon(Icons.spa_outlined, size: 72),

                  const SizedBox(height: 16),

                  const Text(
                    'MasterBook',
                    textAlign: TextAlign.center,
                    style: TextStyle(fontSize: 32, fontWeight: FontWeight.bold),
                  ),

                  const SizedBox(height: 8),

                  const Text(
                    'Онлайн-запись к мастерам',
                    textAlign: TextAlign.center,
                  ),

                  const SizedBox(height: 40),

                  TextField(
                    controller: emailController,
                    keyboardType: TextInputType.emailAddress,
                    decoration: const InputDecoration(
                      labelText: 'Email',
                      border: OutlineInputBorder(),
                      prefixIcon: Icon(Icons.email_outlined),
                    ),
                  ),

                  const SizedBox(height: 16),

                  TextField(
                    controller: passwordController,
                    obscureText: true,
                    decoration: const InputDecoration(
                      labelText: 'Пароль',
                      border: OutlineInputBorder(),
                      prefixIcon: Icon(Icons.lock_outline),
                    ),
                  ),

                  const SizedBox(height: 16),

                  if (errorMessage != null)
                    Text(
                      errorMessage!,
                      textAlign: TextAlign.center,
                      style: const TextStyle(color: Colors.red),
                    ),

                  const SizedBox(height: 16),

                  SizedBox(
                    height: 52,
                    child: FilledButton(
                      onPressed: isLoading ? null : login,
                      child: isLoading
                          ? const SizedBox(
                              width: 24,
                              height: 24,
                              child: CircularProgressIndicator(),
                            )
                          : const Text('Войти'),
                    ),
                  ),

                  const SizedBox(height: 12),

                  OutlinedButton(
                    onPressed: () {
                      Navigator.of(context).push(
                        MaterialPageRoute(
                          builder: (_) => const RegisterScreen(),
                        ),
                      );
                    },
                    child: const Text('Регистрация'),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

// ==================================================
// REGISTER
// ==================================================

class RegisterScreen extends StatefulWidget {
  const RegisterScreen({super.key});

  @override
  State<RegisterScreen> createState() => _RegisterScreenState();
}

class _RegisterScreenState extends State<RegisterScreen> {
  final nameController = TextEditingController();
  final emailController = TextEditingController();
  final passwordController = TextEditingController();
  final confirmPasswordController = TextEditingController();

  bool isLoading = false;
  String? errorMessage;

  Future<void> register() async {
    final name = nameController.text.trim();
    final email = emailController.text.trim();
    final password = passwordController.text;
    final confirmPassword = confirmPasswordController.text;

    if (name.isEmpty) {
      setState(() => errorMessage = 'Введите имя');
      return;
    }

    if (email.isEmpty) {
      setState(() => errorMessage = 'Введите email');
      return;
    }

    if (password.length < 8) {
      setState(() {
        errorMessage = 'Пароль должен содержать минимум 8 символов';
      });
      return;
    }

    if (password != confirmPassword) {
      setState(() => errorMessage = 'Пароли не совпадают');
      return;
    }

    setState(() {
      isLoading = true;
      errorMessage = null;
    });

    try {
      final response = await http.post(
        Uri.parse('$baseUrl/api/v1/auth/register'),
        headers: {'Content-Type': 'application/json'},
        body: jsonEncode({'name': name, 'email': email, 'password': password}),
      );

      final data = jsonDecode(response.body);

      if (response.statusCode != 201) {
        setState(() {
          errorMessage = data['error'] ?? 'Ошибка регистрации';
        });
        return;
      }

      final loginResponse = await http.post(
        Uri.parse('$baseUrl/api/v1/auth/login'),
        headers: {'Content-Type': 'application/json'},
        body: jsonEncode({'email': email, 'password': password}),
      );

      final loginData = jsonDecode(loginResponse.body);

      if (loginResponse.statusCode != 200) {
        setState(() {
          errorMessage = 'Регистрация выполнена, но вход не удался';
        });
        return;
      }

      final prefs = await SharedPreferences.getInstance();

      await prefs.setString('access_token', loginData['access_token']);

      if (!mounted) return;

      Navigator.of(context).pushAndRemoveUntil(
        MaterialPageRoute(builder: (_) => const ClientShell()),
        (route) => false,
      );
    } catch (_) {
      setState(() {
        errorMessage = 'Не удалось подключиться к серверу';
      });
    } finally {
      if (mounted) {
        setState(() {
          isLoading = false;
        });
      }
    }
  }

  @override
  void dispose() {
    nameController.dispose();
    emailController.dispose();
    passwordController.dispose();
    confirmPasswordController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Регистрация')),
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(24),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 420),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  const Text(
                    'Создание аккаунта',
                    textAlign: TextAlign.center,
                    style: TextStyle(fontSize: 28, fontWeight: FontWeight.bold),
                  ),

                  const SizedBox(height: 32),

                  TextField(
                    controller: nameController,
                    decoration: const InputDecoration(
                      labelText: 'Имя',
                      border: OutlineInputBorder(),
                      prefixIcon: Icon(Icons.person_outline),
                    ),
                  ),

                  const SizedBox(height: 16),

                  TextField(
                    controller: emailController,
                    keyboardType: TextInputType.emailAddress,
                    decoration: const InputDecoration(
                      labelText: 'Email',
                      border: OutlineInputBorder(),
                      prefixIcon: Icon(Icons.email_outlined),
                    ),
                  ),

                  const SizedBox(height: 16),

                  TextField(
                    controller: passwordController,
                    obscureText: true,
                    decoration: const InputDecoration(
                      labelText: 'Пароль',
                      border: OutlineInputBorder(),
                      prefixIcon: Icon(Icons.lock_outline),
                    ),
                  ),

                  const SizedBox(height: 16),

                  TextField(
                    controller: confirmPasswordController,
                    obscureText: true,
                    decoration: const InputDecoration(
                      labelText: 'Повторите пароль',
                      border: OutlineInputBorder(),
                      prefixIcon: Icon(Icons.lock_outline),
                    ),
                  ),

                  const SizedBox(height: 16),

                  if (errorMessage != null)
                    Text(
                      errorMessage!,
                      textAlign: TextAlign.center,
                      style: const TextStyle(color: Colors.red),
                    ),

                  const SizedBox(height: 16),

                  SizedBox(
                    height: 52,
                    child: FilledButton(
                      onPressed: isLoading ? null : register,
                      child: isLoading
                          ? const CircularProgressIndicator()
                          : const Text('Зарегистрироваться'),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

// ==================================================
// CLIENT SHELL
// ==================================================

class ClientShell extends StatefulWidget {
  const ClientShell({super.key});

  @override
  State<ClientShell> createState() => _ClientShellState();
}

class _ClientShellState extends State<ClientShell> {
  int currentIndex = 0;

  final pages = const [
    HomeTab(),
    MyAppointmentsScreen(),
    NotificationsScreen(),
    ProfileScreen(),
  ];

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: IndexedStack(index: currentIndex, children: pages),
      bottomNavigationBar: NavigationBar(
        selectedIndex: currentIndex,
        onDestinationSelected: (index) {
          setState(() {
            currentIndex = index;
          });
        },
        destinations: const [
          NavigationDestination(
            icon: Icon(Icons.home_outlined),
            selectedIcon: Icon(Icons.home),
            label: 'Главная',
          ),
          NavigationDestination(
            icon: Icon(Icons.calendar_month_outlined),
            selectedIcon: Icon(Icons.calendar_month),
            label: 'Записи',
          ),
          NavigationDestination(
            icon: Icon(Icons.notifications_outlined),
            selectedIcon: Icon(Icons.notifications),
            label: 'Уведомления',
          ),
          NavigationDestination(
            icon: Icon(Icons.person_outline),
            selectedIcon: Icon(Icons.person),
            label: 'Профиль',
          ),
        ],
      ),
    );
  }
}

// ==================================================
// HOME TAB
// ==================================================

class HomeTab extends StatefulWidget {
  const HomeTab({super.key});

  @override
  State<HomeTab> createState() => _HomeTabState();
}

class _HomeTabState extends State<HomeTab> {
  bool isLoading = true;
  String name = '';

  @override
  void initState() {
    super.initState();
    loadProfile();
  }

  Future<void> loadProfile() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final token = prefs.getString('access_token');

      if (token == null || token.isEmpty) {
        return;
      }

      final response = await http.get(
        Uri.parse('$baseUrl/api/v1/me'),
        headers: {'Authorization': 'Bearer $token'},
      );

      if (response.statusCode == 200) {
        final data = jsonDecode(response.body);

        setState(() {
          name = data['name'] ?? '';
          isLoading = false;
        });

        return;
      }
    } catch (_) {}

    setState(() {
      isLoading = false;
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('MasterBook')),
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          if (isLoading)
            const Center(child: CircularProgressIndicator())
          else
            Text(
              'Добро пожаловать, $name!',
              style: const TextStyle(fontSize: 28, fontWeight: FontWeight.bold),
            ),

          const SizedBox(height: 32),

          Card(
            child: Padding(
              padding: const EdgeInsets.all(20),
              child: Column(
                children: [
                  const Icon(Icons.spa_outlined, size: 56),
                  const SizedBox(height: 12),
                  const Text(
                    'Найдите мастера и запишитесь '
                    'на удобное время.',
                    textAlign: TextAlign.center,
                    style: TextStyle(fontSize: 17),
                  ),
                ],
              ),
            ),
          ),

          const SizedBox(height: 24),

          SizedBox(
            height: 54,
            child: FilledButton.icon(
              onPressed: () {
                Navigator.of(context).push(
                  MaterialPageRoute(builder: (_) => const MastersScreen()),
                );
              },
              icon: const Icon(Icons.people_outline),
              label: const Text('Найти мастера'),
            ),
          ),
        ],
      ),
    );
  }
}

// ==================================================
// MASTERS
// ==================================================

class MastersScreen extends StatefulWidget {
  const MastersScreen({super.key});

  @override
  State<MastersScreen> createState() => _MastersScreenState();
}

class _MastersScreenState extends State<MastersScreen> {
  bool isLoading = true;
  String? errorMessage;
  List<dynamic> masters = [];

  @override
  void initState() {
    super.initState();
    loadMasters();
  }

  Future<void> loadMasters() async {
    try {
      final response = await http.get(Uri.parse('$baseUrl/api/v1/masters'));

      if (response.statusCode == 200) {
        setState(() {
          masters = jsonDecode(response.body);
          isLoading = false;
        });
        return;
      }

      setState(() {
        errorMessage = 'Не удалось загрузить мастеров';
        isLoading = false;
      });
    } catch (_) {
      setState(() {
        errorMessage = 'Не удалось подключиться к серверу';
        isLoading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Мастера')),
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

    if (masters.isEmpty) {
      return const Center(child: Text('Мастеров пока нет'));
    }

    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: masters.length,
      itemBuilder: (context, index) {
        final master = masters[index];

        return Card(
          margin: const EdgeInsets.only(bottom: 12),
          child: ListTile(
            leading: const CircleAvatar(child: Icon(Icons.person)),
            title: Text(
              master['name'] ?? '',
              style: const TextStyle(fontWeight: FontWeight.bold),
            ),
            subtitle: Text(master['description'] ?? ''),
            trailing: const Icon(Icons.chevron_right),
            onTap: () {
              Navigator.of(context).push(
                MaterialPageRoute(
                  builder: (_) => MasterDetailsScreen(
                    masterId: master['id'],
                    masterName: master['name'] ?? '',
                    description: master['description'] ?? '',
                  ),
                ),
              );
            },
          ),
        );
      },
    );
  }
}

// ==================================================
// MASTER DETAILS
// ==================================================

class MasterDetailsScreen extends StatefulWidget {
  final int masterId;
  final String masterName;
  final String description;

  const MasterDetailsScreen({
    super.key,
    required this.masterId,
    required this.masterName,
    required this.description,
  });

  @override
  State<MasterDetailsScreen> createState() => _MasterDetailsScreenState();
}

class _MasterDetailsScreenState extends State<MasterDetailsScreen> {
  bool isLoading = true;
  String? errorMessage;
  List<dynamic> services = [];

  @override
  void initState() {
    super.initState();
    loadServices();
  }

  Future<void> loadServices() async {
    try {
      final response = await http.get(
        Uri.parse(
          '$baseUrl/api/v1/masters/'
          '${widget.masterId}/services',
        ),
      );

      if (response.statusCode == 200) {
        setState(() {
          services = jsonDecode(response.body);
          isLoading = false;
        });
        return;
      }

      setState(() {
        errorMessage = 'Не удалось загрузить услуги';
        isLoading = false;
      });
    } catch (_) {
      setState(() {
        errorMessage = 'Не удалось подключиться к серверу';
        isLoading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    if (isLoading) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }

    if (errorMessage != null) {
      return Scaffold(
        appBar: AppBar(title: Text(widget.masterName)),
        body: Center(child: Text(errorMessage!)),
      );
    }

    return Scaffold(
      appBar: AppBar(title: Text(widget.masterName)),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          CircleAvatar(
            radius: 42,
            child: Text(
              widget.masterName.isNotEmpty
                  ? widget.masterName[0].toUpperCase()
                  : '?',
              style: const TextStyle(fontSize: 30),
            ),
          ),

          const SizedBox(height: 16),

          Text(
            widget.masterName,
            textAlign: TextAlign.center,
            style: const TextStyle(fontSize: 28, fontWeight: FontWeight.bold),
          ),

          const SizedBox(height: 8),

          Text(widget.description, textAlign: TextAlign.center),

          const SizedBox(height: 28),

          const Text(
            'Услуги',
            style: TextStyle(fontSize: 22, fontWeight: FontWeight.bold),
          ),

          const SizedBox(height: 12),

          ...services.map(
            (service) => Card(
              margin: const EdgeInsets.only(bottom: 12),
              child: ListTile(
                title: Text(
                  service['name'] ?? '',
                  style: const TextStyle(fontWeight: FontWeight.bold),
                ),
                subtitle: Text('${service['duration_minutes']} мин'),
                trailing: Text(
                  '${service['price']} ₽',
                  style: const TextStyle(fontWeight: FontWeight.bold),
                ),
                onTap: () {
                  Navigator.of(context).push(
                    MaterialPageRoute(
                      builder: (_) => BookingScreen(
                        masterId: widget.masterId,
                        masterName: widget.masterName,
                        serviceId: service['id'],
                        serviceName: service['name'] ?? '',
                        durationMinutes: service['duration_minutes'] ?? 30,
                        price: service['price'].toString(),
                      ),
                    ),
                  );
                },
              ),
            ),
          ),
        ],
      ),
    );
  }
}

// ==================================================
// BOOKING
// ==================================================

class BookingScreen extends StatefulWidget {
  final int masterId;
  final String masterName;
  final int serviceId;
  final String serviceName;
  final int durationMinutes;
  final String price;

  const BookingScreen({
    super.key,
    required this.masterId,
    required this.masterName,
    required this.serviceId,
    required this.serviceName,
    required this.durationMinutes,
    required this.price,
  });

  @override
  State<BookingScreen> createState() => _BookingScreenState();
}

class _BookingScreenState extends State<BookingScreen> {
  DateTime selectedDate = DateTime.now();
  bool isLoading = false;
  bool isBooking = false;

  String? errorMessage;
  List<dynamic> slots = [];
  String? selectedStartTime;

  @override
  void initState() {
    super.initState();
    loadAvailability();
  }

  String formatDate(DateTime date) {
    return '${date.year}-'
        '${date.month.toString().padLeft(2, '0')}-'
        '${date.day.toString().padLeft(2, '0')}';
  }

  String displayTime(String value) {
    final parsed = DateTime.tryParse(value);

    if (parsed == null) {
      return value;
    }

    return '${parsed.hour.toString().padLeft(2, '0')}:'
        '${parsed.minute.toString().padLeft(2, '0')}';
  }

  Future<void> selectDate() async {
    final result = await showDatePicker(
      context: context,
      initialDate: selectedDate,
      firstDate: DateTime.now(),
      lastDate: DateTime.now().add(const Duration(days: 30)),
    );

    if (result == null) {
      return;
    }

    setState(() {
      selectedDate = result;
      selectedStartTime = null;
    });

    await loadAvailability();
  }

  Future<void> loadAvailability() async {
    setState(() {
      isLoading = true;
      errorMessage = null;
      slots = [];
    });

    try {
      final response = await http.get(
        Uri.parse(
          '$baseUrl/api/v1/masters/'
          '${widget.masterId}/availability'
          '?date=${formatDate(selectedDate)}',
        ),
      );

      if (response.statusCode == 200) {
        setState(() {
          slots = jsonDecode(response.body);
          isLoading = false;
        });
        return;
      }

      setState(() {
        errorMessage = 'Не удалось загрузить свободное время';
        isLoading = false;
      });
    } catch (_) {
      setState(() {
        errorMessage = 'Не удалось подключиться к серверу';
        isLoading = false;
      });
    }
  }

  Future<void> createAppointment() async {
    if (selectedStartTime == null) {
      setState(() {
        errorMessage = 'Выберите время';
      });
      return;
    }

    setState(() {
      isBooking = true;
      errorMessage = null;
    });

    try {
      final prefs = await SharedPreferences.getInstance();
      final token = prefs.getString('access_token');

      if (token == null || token.isEmpty) {
        setState(() {
          errorMessage = 'Необходимо войти в аккаунт';
          isBooking = false;
        });
        return;
      }

      final response = await http.post(
        Uri.parse('$baseUrl/api/v1/appointments'),
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $token',
        },
        body: jsonEncode({
          'master_id': widget.masterId,
          'service_id': widget.serviceId,
          'start_time': selectedStartTime,
        }),
      );

      final data = jsonDecode(response.body);

      if (response.statusCode == 201) {
        if (!mounted) return;

        await showDialog(
          context: context,
          builder: (_) => AlertDialog(
            title: const Text('Запись создана'),
            content: Text('Вы записаны к мастеру ${widget.masterName}.'),
            actions: [
              TextButton(
                onPressed: () {
                  Navigator.of(context).pop();
                },
                child: const Text('OK'),
              ),
            ],
          ),
        );

        if (!mounted) return;

        Navigator.of(context).pop();
        return;
      }

      setState(() {
        errorMessage = data['error'] ?? 'Не удалось создать запись';
      });
    } catch (_) {
      setState(() {
        errorMessage = 'Не удалось подключиться к серверу';
      });
    } finally {
      if (mounted) {
        setState(() {
          isBooking = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Запись')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Text(
            widget.masterName,
            style: const TextStyle(fontSize: 26, fontWeight: FontWeight.bold),
          ),

          const SizedBox(height: 8),

          Text(widget.serviceName, style: const TextStyle(fontSize: 20)),

          const SizedBox(height: 8),

          Text('${widget.durationMinutes} мин • ${widget.price} ₽'),

          const SizedBox(height: 24),

          FilledButton.icon(
            onPressed: selectDate,
            icon: const Icon(Icons.calendar_month),
            label: Text(
              'Дата: '
              '${selectedDate.day.toString().padLeft(2, '0')}.'
              '${selectedDate.month.toString().padLeft(2, '0')}.'
              '${selectedDate.year}',
            ),
          ),

          const SizedBox(height: 24),

          const Text(
            'Свободное время',
            style: TextStyle(fontSize: 22, fontWeight: FontWeight.bold),
          ),

          const SizedBox(height: 12),

          if (isLoading) const Center(child: CircularProgressIndicator()),

          if (!isLoading && errorMessage != null)
            Text(
              errorMessage!,
              textAlign: TextAlign.center,
              style: const TextStyle(color: Colors.red),
            ),

          if (!isLoading && slots.isEmpty)
            const Padding(
              padding: EdgeInsets.all(20),
              child: Text(
                'На выбранную дату '
                'свободного времени нет.',
                textAlign: TextAlign.center,
              ),
            ),

          if (!isLoading && slots.isNotEmpty)
            Wrap(
              spacing: 10,
              runSpacing: 10,
              children: slots.map((slot) {
                final startTime = slot['start_time'] as String;

                return ChoiceChip(
                  label: Text(displayTime(startTime)),
                  selected: selectedStartTime == startTime,
                  onSelected: (_) {
                    setState(() {
                      selectedStartTime = startTime;
                      errorMessage = null;
                    });
                  },
                );
              }).toList(),
            ),

          const SizedBox(height: 32),

          if (selectedStartTime != null)
            SizedBox(
              height: 52,
              child: FilledButton(
                onPressed: isBooking ? null : createAppointment,
                child: isBooking
                    ? const CircularProgressIndicator()
                    : const Text('Записаться', style: TextStyle(fontSize: 16)),
              ),
            ),
        ],
      ),
    );
  }
}

// ==================================================
// PROFILE
// ==================================================

class ProfileScreen extends StatefulWidget {
  const ProfileScreen({super.key});

  @override
  State<ProfileScreen> createState() => _ProfileScreenState();
}

class _ProfileScreenState extends State<ProfileScreen> {
  bool isLoading = true;
  String name = '';
  String email = '';
  String role = '';

  @override
  void initState() {
    super.initState();
    loadProfile();
  }

  Future<void> loadProfile() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final token = prefs.getString('access_token');

      if (token == null || token.isEmpty) {
        return;
      }

      final response = await http.get(
        Uri.parse('$baseUrl/api/v1/me'),
        headers: {'Authorization': 'Bearer $token'},
      );

      if (response.statusCode == 200) {
        final data = jsonDecode(response.body);

        setState(() {
          name = data['name'] ?? '';
          email = data['email'] ?? '';
          role = data['role'] ?? 'client';
          isLoading = false;
        });
      }
    } catch (_) {
      setState(() {
        isLoading = false;
      });
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

  String roleText() {
    switch (role) {
      case 'master':
        return 'Мастер';
      case 'admin':
        return 'Администратор';
      default:
        return 'Клиент';
    }
  }

  @override
  Widget build(BuildContext context) {
    if (isLoading) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }

    return Scaffold(
      appBar: AppBar(title: const Text('Профиль')),
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          const CircleAvatar(radius: 42, child: Icon(Icons.person, size: 44)),

          const SizedBox(height: 20),

          Text(
            name,
            textAlign: TextAlign.center,
            style: const TextStyle(fontSize: 26, fontWeight: FontWeight.bold),
          ),

          const SizedBox(height: 8),

          Text(email, textAlign: TextAlign.center),

          const SizedBox(height: 8),

          Text(roleText(), textAlign: TextAlign.center),

          const SizedBox(height: 32),

          OutlinedButton.icon(
            onPressed: logout,
            icon: const Icon(Icons.logout),
            label: const Text('Выйти'),
          ),
        ],
      ),
    );
  }
}

// ==================================================
// MY APPOINTMENTS
// ==================================================

class MyAppointmentsScreen extends StatefulWidget {
  const MyAppointmentsScreen({super.key});

  @override
  State<MyAppointmentsScreen> createState() => _MyAppointmentsScreenState();
}

class _MyAppointmentsScreenState extends State<MyAppointmentsScreen> {
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
        Uri.parse('$baseUrl/api/v1/appointments'),
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

  Future<void> cancelAppointment(int appointmentId) async {
    final confirm = await showDialog<bool>(
      context: context,
      builder: (_) => AlertDialog(
        title: const Text('Отменить запись?'),
        content: const Text('Вы действительно хотите отменить эту запись?'),
        actions: [
          TextButton(
            onPressed: () {
              Navigator.of(context).pop(false);
            },
            child: const Text('Нет'),
          ),
          FilledButton(
            onPressed: () {
              Navigator.of(context).pop(true);
            },
            child: const Text('Да'),
          ),
        ],
      ),
    );

    if (confirm != true) {
      return;
    }

    try {
      final token = await getToken();

      if (token == null || token.isEmpty) {
        return;
      }

      final response = await http.patch(
        Uri.parse('$baseUrl/api/v1/appointments/$appointmentId/cancel'),
        headers: {'Authorization': 'Bearer $token'},
      );

      if (response.statusCode == 200) {
        if (!mounted) return;

        ScaffoldMessenger.of(context)
            .showSnackBar(const SnackBar(content: Text('Запись отменена')));

        await loadAppointments();
        return;
      }

      if (!mounted) return;

      final data = jsonDecode(response.body);

      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(data['error'] ?? 'Не удалось отменить запись')),
      );
    } catch (_) {
      if (!mounted) return;

      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Не удалось подключиться к серверу')),
      );
    }
  }

  Widget appointmentCard(dynamic appointment) {
    final status = appointment['status'] ?? '';

    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              appointment['master_name'] ?? '',
              style: const TextStyle(fontSize: 20, fontWeight: FontWeight.bold),
            ),

            const SizedBox(height: 8),

            Text(appointment['service_name'] ?? ''),

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

            if (status == 'pending' || status == 'confirmed') ...[
              const SizedBox(height: 12),

              SizedBox(
                width: double.infinity,
                child: OutlinedButton(
                  onPressed: () {
                    cancelAppointment(appointment['id']);
                  },
                  child: const Text('Отменить запись'),
                ),
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
        title: const Text('Мои записи'),
        actions: [
          IconButton(
            onPressed: loadAppointments,
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
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Text(errorMessage!, textAlign: TextAlign.center),
        ),
      );
    }

    if (appointments.isEmpty) {
      return const Center(child: Text('У вас пока нет записей'));
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
