import 'package:flarness_debug/flarness_debug.dart';
import 'package:flutter/material.dart';

void main() {
  FlarnessDebugBinding.ensureInitialized();
  runApp(const MyApp());
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  @override
  Widget build(BuildContext context) {
    const canvas = Color(0xFF0C0E13);
    const panel = Color(0xFF12151D);
    const panelAlt = Color(0xFF171B24);
    const border = Color(0xFF252B38);
    const textPrimary = Color(0xFFF5F7FB);
    const textMuted = Color(0xFF8E97AA);
    const accent = Color(0xFF5E6AD2);
    const accentSoft = Color(0xFF242C4E);
    const success = Color(0xFF31C48D);

    final theme = ThemeData(
      brightness: Brightness.dark,
      useMaterial3: true,
      scaffoldBackgroundColor: canvas,
      colorScheme: const ColorScheme.dark(
        primary: accent,
        secondary: success,
        surface: panel,
      ),
      textTheme: Typography.whiteMountainView.apply(
        bodyColor: textPrimary,
        displayColor: textPrimary,
      ),
      dividerColor: border,
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: panelAlt,
        hintStyle: const TextStyle(color: textMuted),
        labelStyle: const TextStyle(color: textMuted),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(14),
          borderSide: const BorderSide(color: border),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(14),
          borderSide: const BorderSide(color: border),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(14),
          borderSide: const BorderSide(color: accent, width: 1.2),
        ),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          backgroundColor: accent,
          foregroundColor: textPrimary,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(12),
          ),
          padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 14),
        ),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          foregroundColor: textPrimary,
          side: const BorderSide(color: border),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(12),
          ),
        ),
      ),
      chipTheme: ChipThemeData(
        backgroundColor: panelAlt,
        disabledColor: panelAlt,
        selectedColor: accentSoft,
        side: const BorderSide(color: border),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(999)),
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
        labelStyle: const TextStyle(color: textPrimary),
      ),
      cardTheme: CardThemeData(
        color: panel,
        elevation: 0,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(18),
          side: const BorderSide(color: border),
        ),
      ),
    );

    return MaterialApp(
      title: 'Linear Demo',
      debugShowCheckedModeBanner: false,
      theme: theme,
      home: const LinearWorkspacePage(),
    );
  }
}

enum IssueState { backlog, inProgress, done }

enum IssuePriority { low, medium, urgent }

class Issue {
  const Issue({
    required this.id,
    required this.title,
    required this.description,
    required this.team,
    required this.priority,
    required this.state,
  });

  final String id;
  final String title;
  final String description;
  final String team;
  final IssuePriority priority;
  final IssueState state;

  Issue copyWith({
    String? id,
    String? title,
    String? description,
    String? team,
    IssuePriority? priority,
    IssueState? state,
  }) {
    return Issue(
      id: id ?? this.id,
      title: title ?? this.title,
      description: description ?? this.description,
      team: team ?? this.team,
      priority: priority ?? this.priority,
      state: state ?? this.state,
    );
  }
}

class LinearWorkspacePage extends StatefulWidget {
  const LinearWorkspacePage({super.key});

  @override
  State<LinearWorkspacePage> createState() => _LinearWorkspacePageState();
}

class _LinearWorkspacePageState extends State<LinearWorkspacePage> {
  final TextEditingController _searchController = TextEditingController();
  final TextEditingController _titleController = TextEditingController();
  final TextEditingController _descriptionController = TextEditingController();
  final List<Issue> _issues = <Issue>[
    const Issue(
      id: 'FL-101',
      title: 'Polish onboarding command palette',
      description:
          'Refine spacing, hierarchy, and keyboard cues so the command menu feels immediate.',
      team: 'Product',
      priority: IssuePriority.urgent,
      state: IssueState.inProgress,
    ),
    const Issue(
      id: 'FL-102',
      title: 'Tighten daemon restart recovery',
      description:
          'Avoid duplicate daemons and make stop/start transitions deterministic.',
      team: 'Platform',
      priority: IssuePriority.medium,
      state: IssueState.backlog,
    ),
    const Issue(
      id: 'FL-103',
      title: 'Ship semantics tree snapshots',
      description:
          'Capture stable issue snapshots so agents can reason about UI state changes.',
      team: 'Automation',
      priority: IssuePriority.low,
      state: IssueState.done,
    ),
  ];

  IssueState? _activeFilter;
  IssuePriority _draftPriority = IssuePriority.medium;
  bool _composerOpen = true;
  String _statusMessage = 'Inbox synchronized';
  String _selectedIssueId = 'FL-101';
  int _nextIssueNumber = 104;

  @override
  void dispose() {
    _searchController.dispose();
    _titleController.dispose();
    _descriptionController.dispose();
    super.dispose();
  }

  List<Issue> get _visibleIssues {
    final query = _searchController.text.trim().toLowerCase();
    return _issues.where((issue) {
      final matchesFilter =
          _activeFilter == null || issue.state == _activeFilter;
      final matchesSearch =
          query.isEmpty ||
          issue.title.toLowerCase().contains(query) ||
          issue.id.toLowerCase().contains(query) ||
          issue.team.toLowerCase().contains(query);
      return matchesFilter && matchesSearch;
    }).toList();
  }

  Issue get _selectedIssue {
    final visible = _visibleIssues;
    final selected = visible.where((issue) => issue.id == _selectedIssueId);
    if (selected.isNotEmpty) {
      return selected.first;
    }
    if (visible.isNotEmpty) {
      return visible.first;
    }
    return _issues.first;
  }

  int _countFor(IssueState state) {
    return _issues.where((issue) => issue.state == state).length;
  }

  void _openComposer() {
    setState(() {
      _composerOpen = true;
      _statusMessage = 'Drafting a new issue';
    });
  }

  void _closeComposer() {
    setState(() {
      _composerOpen = false;
      _statusMessage = 'Inbox synchronized';
    });
  }

  void _createIssue() {
    final title = _titleController.text.trim();
    if (title.isEmpty) {
      setState(() {
        _statusMessage = 'Issue title is required';
      });
      return;
    }

    final issue = Issue(
      id: 'FL-$_nextIssueNumber',
      title: title,
      description: _descriptionController.text.trim().isEmpty
          ? 'Freshly created from the command surface.'
          : _descriptionController.text.trim(),
      team: 'Automation',
      priority: _draftPriority,
      state: IssueState.backlog,
    );

    setState(() {
      _nextIssueNumber += 1;
      _issues.insert(0, issue);
      _selectedIssueId = issue.id;
      _composerOpen = false;
      _activeFilter = null;
      _titleController.clear();
      _descriptionController.clear();
      _statusMessage = 'Created ${issue.id} ${issue.title}';
    });
  }

  void _moveSelectedIssue(IssueState nextState) {
    final index = _issues.indexWhere((issue) => issue.id == _selectedIssue.id);
    if (index < 0) {
      return;
    }
    final updated = _issues[index].copyWith(state: nextState);
    setState(() {
      _issues[index] = updated;
      _selectedIssueId = updated.id;
      switch (nextState) {
        case IssueState.backlog:
          _statusMessage = 'Reopened ${updated.id} ${updated.title}';
        case IssueState.inProgress:
          _statusMessage =
              'Moved ${updated.id} ${updated.title} to In Progress';
        case IssueState.done:
          _statusMessage = 'Completed ${updated.id} ${updated.title}';
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final size = MediaQuery.sizeOf(context);
    final isDesktop = size.width >= 1100;
    final visibleIssues = _visibleIssues;

    if (!visibleIssues.any((issue) => issue.id == _selectedIssueId) &&
        visibleIssues.isNotEmpty) {
      _selectedIssueId = visibleIssues.first.id;
    }

    return Scaffold(
      body: SafeArea(
        child: Container(
          decoration: const BoxDecoration(
            gradient: LinearGradient(
              colors: <Color>[Color(0xFF0C0E13), Color(0xFF10131B)],
              begin: Alignment.topLeft,
              end: Alignment.bottomRight,
            ),
          ),
          child: Row(
            children: <Widget>[
              if (isDesktop) const _SidebarRail(),
              Expanded(
                child: Padding(
                  padding: const EdgeInsets.all(20),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: <Widget>[
                      _TopBar(
                        searchController: _searchController,
                        onSearchChanged: () => setState(() {}),
                        onCreatePressed: _openComposer,
                      ),
                      const SizedBox(height: 16),
                      _StatusBanner(message: _statusMessage),
                      const SizedBox(height: 16),
                      if (isDesktop)
                        Expanded(
                          child: Row(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: <Widget>[
                              SizedBox(
                                width: 430,
                                child: _IssueListPanel(
                                  boundedHeight: true,
                                  issues: visibleIssues,
                                  selectedIssueId: _selectedIssueId,
                                  activeFilter: _activeFilter,
                                  countFor: _countFor,
                                  onFilterChanged: (filter) {
                                    setState(() {
                                      _activeFilter = filter;
                                    });
                                  },
                                  onIssueSelected: (id) {
                                    setState(() {
                                      _selectedIssueId = id;
                                    });
                                  },
                                ),
                              ),
                              const SizedBox(width: 16),
                              Expanded(
                                child: _IssueDetailPanel(
                                  boundedHeight: true,
                                  issue: _selectedIssue,
                                  onAction: _moveSelectedIssue,
                                ),
                              ),
                              if (isDesktop || _composerOpen) ...<Widget>[
                                const SizedBox(width: 16),
                                SizedBox(
                                  width: 340,
                                  child: _ComposerPanel(
                                    boundedHeight: true,
                                    titleController: _titleController,
                                    descriptionController:
                                        _descriptionController,
                                    selectedPriority: _draftPriority,
                                    onPrioritySelected: (priority) {
                                      setState(() {
                                        _draftPriority = priority;
                                      });
                                    },
                                    onCancel: _closeComposer,
                                    onCreate: _createIssue,
                                  ),
                                ),
                              ],
                            ],
                          ),
                        )
                      else
                        Expanded(
                          child: ListView(
                            children: <Widget>[
                              _IssueListPanel(
                                boundedHeight: false,
                                issues: visibleIssues,
                                selectedIssueId: _selectedIssueId,
                                activeFilter: _activeFilter,
                                countFor: _countFor,
                                onFilterChanged: (filter) {
                                  setState(() {
                                    _activeFilter = filter;
                                  });
                                },
                                onIssueSelected: (id) {
                                  setState(() {
                                    _selectedIssueId = id;
                                  });
                                },
                              ),
                              const SizedBox(height: 16),
                              _IssueDetailPanel(
                                boundedHeight: false,
                                issue: _selectedIssue,
                                onAction: _moveSelectedIssue,
                              ),
                              if (_composerOpen) ...<Widget>[
                                const SizedBox(height: 16),
                                _ComposerPanel(
                                  boundedHeight: false,
                                  titleController: _titleController,
                                  descriptionController: _descriptionController,
                                  selectedPriority: _draftPriority,
                                  onPrioritySelected: (priority) {
                                    setState(() {
                                      _draftPriority = priority;
                                    });
                                  },
                                  onCancel: _closeComposer,
                                  onCreate: _createIssue,
                                ),
                              ],
                            ],
                          ),
                        ),
                    ],
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _SidebarRail extends StatelessWidget {
  const _SidebarRail();

  @override
  Widget build(BuildContext context) {
    const items = <String>['Inbox', 'My issues', 'Projects', 'Cycles', 'Views'];
    return Container(
      width: 220,
      padding: const EdgeInsets.fromLTRB(22, 28, 18, 22),
      decoration: const BoxDecoration(
        color: Color(0xFF0A0C11),
        border: Border(right: BorderSide(color: Color(0xFF252B38))),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          const Text(
            'Flareness',
            style: TextStyle(fontSize: 21, fontWeight: FontWeight.w700),
          ),
          const SizedBox(height: 4),
          const Text(
            'Linear-like debug workspace',
            style: TextStyle(color: Color(0xFF8E97AA)),
          ),
          const SizedBox(height: 24),
          for (final item in items)
            Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: DecoratedBox(
                decoration: BoxDecoration(
                  color: item == 'Inbox'
                      ? const Color(0xFF1A1F2A)
                      : Colors.transparent,
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 12,
                    vertical: 10,
                  ),
                  child: Text(item),
                ),
              ),
            ),
          const Spacer(),
          Container(
            padding: const EdgeInsets.all(14),
            decoration: BoxDecoration(
              color: const Color(0xFF12151D),
              borderRadius: BorderRadius.circular(16),
              border: Border.all(color: const Color(0xFF252B38)),
            ),
            child: const Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                Text('Today', style: TextStyle(fontWeight: FontWeight.w600)),
                SizedBox(height: 8),
                Text(
                  'Ship generic Flutter text input and make daemon reuse deterministic.',
                  style: TextStyle(color: Color(0xFF8E97AA), height: 1.45),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _TopBar extends StatelessWidget {
  const _TopBar({
    required this.searchController,
    required this.onSearchChanged,
    required this.onCreatePressed,
  });

  final TextEditingController searchController;
  final VoidCallback onSearchChanged;
  final VoidCallback onCreatePressed;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: <Widget>[
        Expanded(
          child: TextField(
            key: const ValueKey<String>('search-input'),
            controller: searchController,
            onChanged: (_) => onSearchChanged(),
            decoration: const InputDecoration(
              prefixIcon: Icon(Icons.search, size: 18),
              labelText: 'Search issues',
              hintText: 'Search by id, team, or title',
            ),
          ),
        ),
        const SizedBox(width: 12),
        OutlinedButton(onPressed: () {}, child: const Text('⌘K Command menu')),
        const SizedBox(width: 12),
        FilledButton(
          key: const ValueKey<String>('open-create-issue-button'),
          onPressed: onCreatePressed,
          child: const Text('New issue'),
        ),
      ],
    );
  }
}

class _StatusBanner extends StatelessWidget {
  const _StatusBanner({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    return Container(
      key: const ValueKey<String>('status-banner'),
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
      decoration: BoxDecoration(
        color: const Color(0xFF141925),
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: const Color(0xFF252B38)),
      ),
      child: Text(message, style: const TextStyle(fontWeight: FontWeight.w600)),
    );
  }
}

class _IssueListPanel extends StatelessWidget {
  const _IssueListPanel({
    required this.boundedHeight,
    required this.issues,
    required this.selectedIssueId,
    required this.activeFilter,
    required this.countFor,
    required this.onFilterChanged,
    required this.onIssueSelected,
  });

  final bool boundedHeight;
  final List<Issue> issues;
  final String selectedIssueId;
  final IssueState? activeFilter;
  final int Function(IssueState state) countFor;
  final ValueChanged<IssueState?> onFilterChanged;
  final ValueChanged<String> onIssueSelected;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(18),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            const Text(
              'Inbox',
              style: TextStyle(fontSize: 24, fontWeight: FontWeight.w700),
            ),
            const SizedBox(height: 6),
            const Text(
              'Dense list interactions, shallow hierarchy, and one selected issue at a time.',
              style: TextStyle(color: Color(0xFF8E97AA), height: 1.45),
            ),
            const SizedBox(height: 16),
            Wrap(
              spacing: 10,
              runSpacing: 10,
              children: <Widget>[
                _CountChip(
                  label: 'All',
                  count: issues.length.toString(),
                  selected: activeFilter == null,
                  onTap: () => onFilterChanged(null),
                ),
                for (final state in IssueState.values)
                  _CountChip(
                    key: ValueKey<String>('filter-${state.name}'),
                    label: _stateLabel(state),
                    count: countFor(state).toString(),
                    selected: activeFilter == state,
                    onTap: () => onFilterChanged(state),
                  ),
              ],
            ),
            const SizedBox(height: 16),
            if (boundedHeight) Expanded(child: _buildList()) else _buildList(),
          ],
        ),
      ),
    );
  }

  Widget _buildList() {
    if (issues.isEmpty) {
      return const Padding(
        padding: EdgeInsets.symmetric(vertical: 28),
        child: Center(
          child: Text(
            'No issues match the current query.',
            style: TextStyle(color: Color(0xFF8E97AA)),
          ),
        ),
      );
    }

    return ListView.separated(
      shrinkWrap: !boundedHeight,
      physics: boundedHeight
          ? const AlwaysScrollableScrollPhysics()
          : const NeverScrollableScrollPhysics(),
      itemCount: issues.length,
      separatorBuilder: (_, _) => const SizedBox(height: 10),
      itemBuilder: (context, index) {
        final issue = issues[index];
        return _IssueRow(
          issue: issue,
          selected: issue.id == selectedIssueId,
          onTap: () => onIssueSelected(issue.id),
        );
      },
    );
  }
}

class _IssueRow extends StatelessWidget {
  const _IssueRow({
    required this.issue,
    required this.selected,
    required this.onTap,
  });

  final Issue issue;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final borderColor = selected
        ? const Color(0xFF5E6AD2)
        : const Color(0xFF252B38);
    final background = selected
        ? const Color(0xFF181F34)
        : const Color(0xFF10131A);

    return InkWell(
      key: ValueKey<String>('issue-row-${issue.id}'),
      onTap: onTap,
      borderRadius: BorderRadius.circular(16),
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 160),
        padding: const EdgeInsets.all(14),
        decoration: BoxDecoration(
          color: background,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(color: borderColor),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            Row(
              children: <Widget>[
                Text(
                  issue.id,
                  style: const TextStyle(
                    color: Color(0xFF8E97AA),
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const SizedBox(width: 8),
                _TinyPill(label: issue.team),
                const Spacer(),
                _StateDot(state: issue.state),
              ],
            ),
            const SizedBox(height: 10),
            Text(
              issue.title,
              style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w700),
            ),
            const SizedBox(height: 6),
            Text(
              issue.description,
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(color: Color(0xFF8E97AA), height: 1.45),
            ),
          ],
        ),
      ),
    );
  }
}

class _IssueDetailPanel extends StatelessWidget {
  const _IssueDetailPanel({
    required this.issue,
    required this.boundedHeight,
    required this.onAction,
  });

  final bool boundedHeight;
  final Issue issue;
  final ValueChanged<IssueState> onAction;

  @override
  Widget build(BuildContext context) {
    final nextAction = switch (issue.state) {
      IssueState.backlog => (
        label: 'Start ${issue.title}',
        state: IssueState.inProgress,
      ),
      IssueState.inProgress => (
        label: 'Complete ${issue.title}',
        state: IssueState.done,
      ),
      IssueState.done => (
        label: 'Reopen ${issue.title}',
        state: IssueState.backlog,
      ),
    };

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(22),
        child: Column(
          mainAxisSize: boundedHeight ? MainAxisSize.max : MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            Row(
              children: <Widget>[
                Text(
                  issue.id,
                  style: const TextStyle(
                    color: Color(0xFF8E97AA),
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const SizedBox(width: 10),
                _TinyPill(label: issue.team),
                const SizedBox(width: 8),
                _TinyPill(label: _priorityLabel(issue.priority)),
                const Spacer(),
                _TinyPill(label: _stateLabel(issue.state)),
              ],
            ),
            const SizedBox(height: 18),
            Text(
              issue.title,
              style: const TextStyle(
                fontSize: 30,
                fontWeight: FontWeight.w700,
                letterSpacing: -0.8,
              ),
            ),
            const SizedBox(height: 12),
            Text(
              issue.description,
              style: const TextStyle(
                color: Color(0xFFA0A8B9),
                height: 1.6,
                fontSize: 15,
              ),
            ),
            const SizedBox(height: 22),
            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(16),
              decoration: BoxDecoration(
                color: const Color(0xFF10141C),
                borderRadius: BorderRadius.circular(16),
                border: Border.all(color: const Color(0xFF252B38)),
              ),
              child: const Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: <Widget>[
                  Text(
                    'Activity',
                    style: TextStyle(fontWeight: FontWeight.w600),
                  ),
                  SizedBox(height: 10),
                  Text(
                    'Automation-friendly issue details live here: the selected issue owns the primary actions and current state.',
                    style: TextStyle(color: Color(0xFF8E97AA), height: 1.5),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 22),
            Row(
              children: <Widget>[
                FilledButton(
                  onPressed: () => onAction(nextAction.state),
                  child: Text(nextAction.label),
                ),
                const SizedBox(width: 12),
                OutlinedButton(onPressed: () {}, child: const Text('Assign')),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _ComposerPanel extends StatelessWidget {
  const _ComposerPanel({
    required this.titleController,
    required this.descriptionController,
    required this.boundedHeight,
    required this.selectedPriority,
    required this.onPrioritySelected,
    required this.onCancel,
    required this.onCreate,
  });

  final TextEditingController titleController;
  final TextEditingController descriptionController;
  final bool boundedHeight;
  final IssuePriority selectedPriority;
  final ValueChanged<IssuePriority> onPrioritySelected;
  final VoidCallback onCancel;
  final VoidCallback onCreate;

  @override
  Widget build(BuildContext context) {
    final content = Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        Row(
          children: <Widget>[
            const Text(
              'New issue',
              style: TextStyle(fontSize: 20, fontWeight: FontWeight.w700),
            ),
            const Spacer(),
            IconButton(
              key: const ValueKey<String>('close-create-issue-button'),
              onPressed: onCancel,
              icon: const Icon(Icons.close, size: 18),
            ),
          ],
        ),
        const SizedBox(height: 6),
        const Text(
          'Short, structured, and optimized for fast capture.',
          style: TextStyle(color: Color(0xFF8E97AA), height: 1.45),
        ),
        const SizedBox(height: 16),
        TextField(
          key: const ValueKey<String>('issue-title-input'),
          controller: titleController,
          decoration: const InputDecoration(
            labelText: 'Issue title',
            hintText: 'Describe the problem clearly',
          ),
        ),
        const SizedBox(height: 16),
        const Text(
          'Priority',
          style: TextStyle(fontWeight: FontWeight.w600),
        ),
        const SizedBox(height: 10),
        Wrap(
          spacing: 10,
          runSpacing: 10,
          children: IssuePriority.values.map((priority) {
            return ChoiceChip(
              key: ValueKey<String>('priority-${priority.name}'),
              label: Text(_priorityLabel(priority)),
              selected: selectedPriority == priority,
              onSelected: (_) => onPrioritySelected(priority),
            );
          }).toList(),
        ),
        const SizedBox(height: 20),
        SizedBox(
          width: double.infinity,
          child: FilledButton(
            key: const ValueKey<String>('create-issue-button'),
            onPressed: onCreate,
            child: const Text('Create issue'),
          ),
        ),
        const SizedBox(height: 16),
        TextField(
          key: const ValueKey<String>('issue-description-input'),
          controller: descriptionController,
          maxLines: 4,
          decoration: const InputDecoration(
            labelText: 'Description',
            hintText: 'Add context, acceptance criteria, or follow-up notes',
          ),
        ),
      ],
    );

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(18),
        child: boundedHeight
            ? SingleChildScrollView(child: content)
            : content,
      ),
    );
  }
}

class _CountChip extends StatelessWidget {
  const _CountChip({
    super.key,
    required this.label,
    required this.count,
    required this.selected,
    required this.onTap,
  });
  final String label;
  final String count;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(999),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        decoration: BoxDecoration(
          color: selected ? const Color(0xFF222B46) : const Color(0xFF161A22),
          borderRadius: BorderRadius.circular(999),
          border: Border.all(color: const Color(0xFF252B38)),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: <Widget>[
            Text(label),
            const SizedBox(width: 8),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
              decoration: BoxDecoration(
                color: const Color(0xFF0F131B),
                borderRadius: BorderRadius.circular(999),
              ),
              child: Text(
                count,
                style: const TextStyle(color: Color(0xFF8E97AA), fontSize: 12),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _TinyPill extends StatelessWidget {
  const _TinyPill({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 9, vertical: 5),
      decoration: BoxDecoration(
        color: const Color(0xFF1A1F2A),
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: const Color(0xFF252B38)),
      ),
      child: Text(
        label,
        style: const TextStyle(fontSize: 12, color: Color(0xFFE6EBF7)),
      ),
    );
  }
}

class _StateDot extends StatelessWidget {
  const _StateDot({required this.state});

  final IssueState state;

  @override
  Widget build(BuildContext context) {
    final color = switch (state) {
      IssueState.backlog => const Color(0xFF8E97AA),
      IssueState.inProgress => const Color(0xFF5E6AD2),
      IssueState.done => const Color(0xFF31C48D),
    };
    return Container(
      width: 10,
      height: 10,
      decoration: BoxDecoration(color: color, shape: BoxShape.circle),
    );
  }
}

String _stateLabel(IssueState state) {
  return switch (state) {
    IssueState.backlog => 'Backlog',
    IssueState.inProgress => 'In Progress',
    IssueState.done => 'Done',
  };
}

String _priorityLabel(IssuePriority priority) {
  return switch (priority) {
    IssuePriority.low => 'Low',
    IssuePriority.medium => 'Medium',
    IssuePriority.urgent => 'Urgent',
  };
}
