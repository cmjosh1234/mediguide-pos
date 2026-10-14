part of '../screens/publication_guideline_page.dart';

class _GuidelineTabs extends StatelessWidget {
  const _GuidelineTabs({
    required this.selectedIndex,
    this.contentsLabel = 'Chapters',
    required this.hasChapters,
    required this.hasKeyPoints,
    required this.hasTables,
    required this.onSelected,
  });

  final int selectedIndex;
  final String contentsLabel;

  final bool hasChapters;
  final bool hasKeyPoints;
  final bool hasTables;

  final ValueChanged<int> onSelected;

  @override
  Widget build(BuildContext context) {
    final tabs = <(String, bool)>[
      ('Overview', true),
      (contentsLabel, hasChapters),
      ('Key Points', hasKeyPoints),
      ('Tables', hasTables),
    ];

    return DecoratedBox(
      decoration: BoxDecoration(
        border: Border(
          bottom: BorderSide(color: Theme.of(context).dividerColor),
        ),
      ),
      child: SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        child: Row(
          children: [
            for (var index = 0; index < tabs.length; index++)
              if (tabs[index].$2)
                Semantics(
                  selected: selectedIndex == index,
                  button: true,
                  child: InkWell(
                    onTap: tabs[index].$2
                        ? () {
                            onSelected(index);
                          }
                        : null,
                    child: Container(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 14,
                        vertical: 12,
                      ),
                      decoration: BoxDecoration(
                        border: Border(
                          bottom: BorderSide(
                            width: 2,
                            color: selectedIndex == index
                                ? Theme.of(context).colorScheme.primary
                                : Colors.transparent,
                          ),
                        ),
                      ),
                      child: Text(
                        tabs[index].$1,
                        style: Theme.of(context).textTheme.labelLarge?.copyWith(
                          color: !tabs[index].$2
                              ? Theme.of(context).disabledColor
                              : selectedIndex == index
                              ? Theme.of(context).colorScheme.primary
                              : null,
                          fontWeight: selectedIndex == index
                              ? FontWeight.w700
                              : null,
                        ),
                      ),
                    ),
                  ),
                ),
          ],
        ),
      ),
    );
  }
}

// =============================================================================
// ABOUT
// =============================================================================
