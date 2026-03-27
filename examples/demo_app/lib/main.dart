import 'package:flarness_debug/flarness_debug.dart';
import 'package:flutter/material.dart';

void main() {
  FlarnessDebugBinding.ensureInitialized();
  runApp(const MyApp());
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  // This widget is the root of your application.
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Flutter Demo',
      theme: ThemeData(
        // This is the theme of your application.
        //
        // TRY THIS: Try running your application with "flutter run". You'll see
        // the application has a purple toolbar. Then, without quitting the app,
        // try changing the seedColor in the colorScheme below to Colors.green
        // and then invoke "hot reload" (save your changes or press the "hot
        // reload" button in a Flutter-supported IDE, or press "r" if you used
        // the command line to start the app).
        //
        // Notice that the counter didn't reset back to zero; the application
        // state is not lost during the reload. To reset the state, use hot
        // restart instead.
        //
        // This works for code too, not just values: Most code changes can be
        // tested with just a hot reload.
        colorScheme: .fromSeed(seedColor: Colors.deepPurple),
      ),
      home: const MyHomePage(title: 'Flutter Demo Home Page'),
    );
  }
}

class MyHomePage extends StatefulWidget {
  const MyHomePage({super.key, required this.title});

  // This widget is the home page of your application. It is stateful, meaning
  // that it has a State object (defined below) that contains fields that affect
  // how it looks.

  // This class is the configuration for the state. It holds the values (in this
  // case the title) provided by the parent (in this case the App widget) and
  // used by the build method of the State. Fields in a Widget subclass are
  // always marked "final".

  final String title;

  @override
  State<MyHomePage> createState() => _MyHomePageState();
}

class _MyHomePageState extends State<MyHomePage> {
  final TextEditingController _controller = TextEditingController();
  final List<String> _items = List<String>.generate(
    30,
    (int index) => 'Task ${index + 1}',
  );

  int _counter = 0;
  String _status = 'Idle';

  void _incrementCounter() {
    setState(() {
      _counter++;
      _status = 'Counter updated to $_counter';
    });
  }

  void _submit() {
    setState(() {
      final String value = _controller.text.trim();
      _status = value.isEmpty ? 'Nothing submitted' : 'Submitted: $value';
      if (value.isNotEmpty) {
        _items.insert(0, value);
        _controller.clear();
      }
    });
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        backgroundColor: Theme.of(context).colorScheme.inversePrimary,
        title: Text(widget.title),
      ),
      body: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            TextField(
              key: const ValueKey<String>('demo-input'),
              controller: _controller,
              decoration: const InputDecoration(
                labelText: 'Task Input',
                hintText: 'Type a new task',
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 12),
            FilledButton(
              key: const ValueKey<String>('submit-button'),
              onPressed: _submit,
              child: const Text('Submit Task'),
            ),
            const SizedBox(height: 16),
            Text(
              _status,
              key: const ValueKey<String>('status-text'),
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 16),
            Text(
              'Counter: $_counter',
              key: const ValueKey<String>('counter-text'),
              style: Theme.of(context).textTheme.headlineSmall,
            ),
            const SizedBox(height: 16),
            Expanded(
              child: ListView.builder(
                key: const ValueKey<String>('task-list'),
                itemCount: _items.length,
                itemBuilder: (BuildContext context, int index) {
                  return ListTile(
                    title: Text(_items[index]),
                    subtitle: Text('Row ${index + 1}'),
                  );
                },
              ),
            ),
          ],
        ),
      ),
      floatingActionButton: FloatingActionButton(
        key: const ValueKey<String>('increment-button'),
        onPressed: _incrementCounter,
        tooltip: 'Increment',
        child: const Icon(Icons.add),
      ),
    );
  }
}
