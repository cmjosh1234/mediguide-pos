import 'package:user_app/features/guidelines/data/models/guideline_publication.dart';

/// Presentation is independent of extraction quality. Publication flags always
/// win over the kind's slug, including for administrator-created kinds.
enum DocumentReaderKind {
  guideline,
  procedure,
  checklist,
  caseDefinition,
  training,
  reference,
  uploaded,
  link;

  static DocumentReaderKind forPublication(GuidelinePublication publication) {
    final kind = publication.documentKind;
    if (kind?.publishAsLink == true) return link;
    if (kind?.publishAsUploaded == true) return uploaded;
    return switch (kind?.slug.trim().toLowerCase().replaceAll('-', '_')) {
      'manual' => guideline,
      'form' => uploaded,
      'sop' ||
      'ipc_protocol' ||
      'laboratory_protocol' ||
      'surveillance_protocol' ||
      'contact_tracing_guide' ||
      'treatment_protocol' ||
      'referral_protocol' => procedure,
      'checklist' => checklist,
      'case_definition' => caseDefinition,
      'training_material' => training,
      'framework' ||
      'communication_material' ||
      'policy' ||
      'situation_report_attachment' => reference,
      _ => guideline,
    };
  }

  String get label => switch (this) {
    guideline => 'Guideline reader',
    procedure => 'Procedure reader',
    checklist => 'Checklist reader',
    caseDefinition => 'Case definition reader',
    training => 'Training reader',
    reference => 'Reference reader',
    uploaded => 'Uploaded document',
    link => 'External website',
  };
}
