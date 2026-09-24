import {
  Activity,
  Baby,
  BadgeAlert,
  Bandage,
  Bean,
  Biohazard,
  Bone,
  Brain,
  Bug,
  Calculator,
  CalendarHeart,
  CircleHelp,
  Dna,
  Dog,
  Droplet,
  Droplets,
  Ear,
  Eye,
  FlaskConical,
  Hand,
  HandHeart,
  Heart,
  HeartPulse,
  LayoutGrid,
  ListChecks,
  type LucideIcon,
  Microscope,
  Pill,
  Rat,
  Ribbon,
  Salad,
  ShieldAlert,
  ShieldCheck,
  ShieldPlus,
  Siren,
  Skull,
  Stethoscope,
  Syringe,
  Thermometer,
  Virus,
  Wind,
  Worm,
} from "lucide-react";

// Tones group icons into clinical families so the grid reads at a glance.
type Tone =
  | "vector"
  | "outbreak"
  | "water"
  | "cardio"
  | "respiratory"
  | "prevention"
  | "maternal"
  | "mind"
  | "neutral";

type IconEntry = { icon: LucideIcon; tone: Tone };

// Lucide names the backend may store, plus aliases for names Lucide lacks.
const NAMED_ICONS: Record<string, IconEntry> = {
  activity: { icon: Activity, tone: "cardio" },
  baby: { icon: Baby, tone: "maternal" },
  "badge-alert": { icon: BadgeAlert, tone: "outbreak" },
  bandage: { icon: Bandage, tone: "neutral" },
  biohazard: { icon: Biohazard, tone: "outbreak" },
  brain: { icon: Brain, tone: "mind" },
  bug: { icon: Bug, tone: "vector" },
  calculator: { icon: Calculator, tone: "neutral" },
  "calendar-heart": { icon: CalendarHeart, tone: "maternal" },
  "circle-help": { icon: CircleHelp, tone: "neutral" },
  "help-circle": { icon: CircleHelp, tone: "neutral" },
  droplet: { icon: Droplet, tone: "water" },
  droplets: { icon: Droplets, tone: "water" },
  "hand-heart": { icon: HandHeart, tone: "maternal" },
  heart: { icon: Heart, tone: "cardio" },
  "heart-pulse": { icon: HeartPulse, tone: "cardio" },
  "layout-grid": { icon: LayoutGrid, tone: "neutral" },
  "list-checks": { icon: ListChecks, tone: "neutral" },
  microscope: { icon: Microscope, tone: "neutral" },
  mosquito: { icon: Bug, tone: "vector" },
  pill: { icon: Pill, tone: "neutral" },
  "shield-alert": { icon: ShieldAlert, tone: "outbreak" },
  "shield-check": { icon: ShieldCheck, tone: "prevention" },
  "shield-plus": { icon: ShieldPlus, tone: "prevention" },
  siren: { icon: Siren, tone: "outbreak" },
  stethoscope: { icon: Stethoscope, tone: "neutral" },
  syringe: { icon: Syringe, tone: "prevention" },
  thermometer: { icon: Thermometer, tone: "outbreak" },
  virus: { icon: Virus, tone: "outbreak" },
  wind: { icon: Wind, tone: "respiratory" },
  worm: { icon: Worm, tone: "vector" },
};

// Checked in order against the name/slug; the first match wins, so a more
// specific disease (e.g. Ebola) overrides a stored generic shield icon.
const KEYWORD_ICONS: Array<[RegExp, IconEntry]> = [
  [/malaria|mosquito|dengue|yellow fever|trypano|sleeping sickness/, { icon: Bug, tone: "vector" }],
  [/ebola|marburg|haemorrhagic|hemorrhagic|anthrax|mpox|monkeypox/, { icon: Biohazard, tone: "outbreak" }],
  [/cholera|diarrh|dysentery|typhoid|hepatitis a|water/, { icon: Droplets, tone: "water" }],
  [/plague/, { icon: Rat, tone: "outbreak" }],
  [/rabies|dog bite|animal bite/, { icon: Dog, tone: "outbreak" }],
  [/covid|influenza|\bflu\b|viral/, { icon: Virus, tone: "outbreak" }],
  [/hiv|aids|cancer|oncolog/, { icon: Ribbon, tone: "maternal" }],
  [/measles|polio|rubella|diphtheria|pertussis|tetanus|vaccin|immuni/, { icon: Syringe, tone: "prevention" }],
  [/tubercul|\btb\b|pneumon|asthma|copd|respirat|lung/, { icon: Wind, tone: "respiratory" }],
  [/hypertens|cardi|heart|stroke/, { icon: HeartPulse, tone: "cardio" }],
  [/diabet|glucose|anaemia|anemia|blood/, { icon: Droplet, tone: "cardio" }],
  [/sickle|genetic/, { icon: Dna, tone: "cardio" }],
  [/kidney|renal/, { icon: Bean, tone: "cardio" }],
  [/matern|pregnan|neonat|newborn|child|paediatric|pediatric|obstet/, { icon: Baby, tone: "maternal" }],
  [/mental|depress|psych|epilep|seizure|neuro/, { icon: Brain, tone: "mind" }],
  [/worm|helminth|schisto|bilharz/, { icon: Worm, tone: "vector" }],
  [/nutrition|malnutri|obesity/, { icon: Salad, tone: "prevention" }],
  [/eye|ophthalm|trachoma/, { icon: Eye, tone: "neutral" }],
  [/\bear\b|otitis/, { icon: Ear, tone: "neutral" }],
  [/skin|dermat|leprosy|scabies/, { icon: Hand, tone: "neutral" }],
  [/fracture|bone|osteo/, { icon: Bone, tone: "neutral" }],
  [/injur|trauma|burn|wound/, { icon: Bandage, tone: "neutral" }],
  [/poison|snake|venom/, { icon: Skull, tone: "outbreak" }],
  [/fever/, { icon: Thermometer, tone: "outbreak" }],
  [/\blab\b|diagnos|\btest/, { icon: FlaskConical, tone: "neutral" }],
];

const FALLBACK: IconEntry = { icon: Stethoscope, tone: "neutral" };

function resolveTaxonomyIcon(
  icon?: string | null,
  label?: string | null,
): IconEntry {
  const text = (label ?? "").toLowerCase().replace(/[-_]/g, " ");
  const byKeyword = text
    ? KEYWORD_ICONS.find(([pattern]) => pattern.test(text))?.[1]
    : undefined;
  const named = icon ? NAMED_ICONS[icon.trim().toLowerCase()] : undefined;
  return byKeyword ?? named ?? FALLBACK;
}

export function TaxonomyIcon({
  icon,
  label,
  className = "resource-icon",
}: {
  icon?: string | null;
  label?: string | null;
  className?: string;
}) {
  const { icon: Icon, tone } = resolveTaxonomyIcon(icon, label);
  return (
    <span className={className} data-tone={tone} aria-hidden="true">
      <Icon size={24} strokeWidth={1.8} />
    </span>
  );
}
