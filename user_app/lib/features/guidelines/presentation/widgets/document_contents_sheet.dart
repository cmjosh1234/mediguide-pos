import 'package:flutter/material.dart';
import 'package:user_app/features/guidelines/data/models/guideline_publication.dart';

class DocumentContentsSheet extends StatefulWidget {
  const DocumentContentsSheet({
    super.key,
    required this.sections,
    required this.label,
    this.selectedId,
  });
  final List<PublicationSection> sections;
  final String label;
  final String? selectedId;
  @override
  State<DocumentContentsSheet> createState() => _DocumentContentsSheetState();
}

class _DocumentContentsSheetState extends State<DocumentContentsSheet> {
  final _search = TextEditingController();
  String _query = '';
  @override
  void dispose() {
    _search.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final matches = widget.sections
        .where(
          (section) =>
              section.title.toLowerCase().contains(_query.toLowerCase().trim()),
        )
        .toList();
    return Padding(
      padding: EdgeInsets.only(bottom: MediaQuery.viewInsetsOf(context).bottom),
      child: SizedBox(
        height: MediaQuery.sizeOf(context).height * 0.78,
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(24, 8, 8, 0),
              child: Row(
                children: [
                  Expanded(
                    child: Text(
                      widget.label,
                      style: Theme.of(context).textTheme.titleLarge,
                    ),
                  ),
                  IconButton(
                    tooltip: 'Close contents',
                    onPressed: () => Navigator.pop(context),
                    icon: const Icon(Icons.close),
                  ),
                ],
              ),
            ),
            Padding(
              padding: const EdgeInsets.all(16),
              child: TextField(
                controller: _search,
                onChanged: (query) => setState(() => _query = query),
                decoration: InputDecoration(
                  labelText: 'Find a section',
                  prefixIcon: const Icon(Icons.search),
                  suffixIcon: _query.isEmpty
                      ? null
                      : IconButton(
                          tooltip: 'Clear search',
                          icon: const Icon(Icons.close),
                          onPressed: () {
                            _search.clear();
                            setState(() => _query = '');
                          },
                        ),
                  border: const OutlineInputBorder(),
                ),
              ),
            ),
            Expanded(
              child: matches.isEmpty
                  ? const Center(
                      child: Padding(
                        padding: EdgeInsets.all(24),
                        child: Text(
                          'No matching sections. Try another search.',
                        ),
                      ),
                    )
                  : ListView.builder(
                      itemCount: matches.length,
                      itemBuilder: (_, index) {
                        final section = matches[index];
                        final selected = section.id == widget.selectedId;
                        return ListTile(
                          selected: selected,
                          contentPadding: EdgeInsets.fromLTRB(
                            16 + (section.level - 1).clamp(0, 4) * 12.0,
                            4,
                            16,
                            4,
                          ),
                          leading: Icon(
                            selected
                                ? Icons.radio_button_checked
                                : Icons.article_outlined,
                          ),
                          title: Text(section.title),
                          subtitle: section.pageLabel.isEmpty
                              ? null
                              : Text(section.pageLabel),
                          trailing: const Icon(Icons.chevron_right),
                          onTap: () => Navigator.pop(context, section.id),
                        );
                      },
                    ),
            ),
          ],
        ),
      ),
    );
  }
}
