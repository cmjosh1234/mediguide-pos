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
  ClipboardCheck,
  ClipboardList,
  Dna,
  Droplet,
  Droplets,
  Ear,
  Eye,
  FlaskConical,
  Gauge,
  Hand,
  HandHeart,
  Heart,
  HeartPulse,
  ListChecks,
  type LucideIcon,
  Microscope,
  PersonStanding,
  Pill,
  Ribbon,
  Ruler,
  Salad,
  ShieldAlert,
  ShieldCheck,
  ShieldPlus,
  Siren,
  Skull,
  Stethoscope,
  Syringe,
  Thermometer,
  Timer,
  Weight,
  Wind,
  Worm,
} from "lucide-react"

export type ClinicalIcon = {
  /** Lucide icon name, the value stored on the record. */
  name: string
  label: string
  /** Extra search terms. */
  keywords: string
  icon: LucideIcon
}

/**
 * Curated Lucide icons offered for clinical content. Names match the ones the
 * guidelines platform's TaxonomyIcon resolves, so stored values render there too.
 */
export const clinicalIcons: ClinicalIcon[] = [
  { name: "calculator", label: "Calculator", keywords: "dose score", icon: Calculator },
  { name: "list-checks", label: "Checklist", keywords: "tasks steps", icon: ListChecks },
  { name: "clipboard-list", label: "Clipboard", keywords: "assessment form", icon: ClipboardList },
  { name: "clipboard-check", label: "Clipboard check", keywords: "assessment done", icon: ClipboardCheck },
  { name: "gauge", label: "Gauge", keywords: "score scale severity", icon: Gauge },
  { name: "timer", label: "Timer", keywords: "time duration", icon: Timer },
  { name: "weight", label: "Weight", keywords: "bmi mass", icon: Weight },
  { name: "ruler", label: "Ruler", keywords: "height length measure", icon: Ruler },
  { name: "stethoscope", label: "Stethoscope", keywords: "clinical exam general", icon: Stethoscope },
  { name: "activity", label: "Activity", keywords: "vitals pulse bmi", icon: Activity },
  { name: "heart", label: "Heart", keywords: "cardiac", icon: Heart },
  { name: "heart-pulse", label: "Heart pulse", keywords: "blood pressure hypertension cardiac", icon: HeartPulse },
  { name: "droplet", label: "Droplet", keywords: "blood dehydration glucose", icon: Droplet },
  { name: "droplets", label: "Droplets", keywords: "fluid balance water cholera", icon: Droplets },
  { name: "brain", label: "Brain", keywords: "neuro coma mental", icon: Brain },
  { name: "baby", label: "Baby", keywords: "newborn neonatal paediatric pediatric apgar", icon: Baby },
  { name: "calendar-heart", label: "Calendar heart", keywords: "pregnancy due date antenatal", icon: CalendarHeart },
  { name: "hand-heart", label: "Hand heart", keywords: "care maternal support", icon: HandHeart },
  { name: "person-standing", label: "Person standing", keywords: "maternal woman adult patient", icon: PersonStanding },
  { name: "thermometer", label: "Thermometer", keywords: "fever temperature", icon: Thermometer },
  { name: "siren", label: "Siren", keywords: "emergency triage urgent", icon: Siren },
  { name: "badge-alert", label: "Alert badge", keywords: "pain warning", icon: BadgeAlert },
  { name: "bandage", label: "Bandage", keywords: "wound injury trauma burn", icon: Bandage },
  { name: "bone", label: "Bone", keywords: "fracture orthopaedic", icon: Bone },
  { name: "pill", label: "Pill", keywords: "medication drug dosage", icon: Pill },
  { name: "syringe", label: "Syringe", keywords: "injection vaccine immunization", icon: Syringe },
  { name: "flask-conical", label: "Flask", keywords: "lab test diagnostic", icon: FlaskConical },
  { name: "microscope", label: "Microscope", keywords: "lab diagnosis", icon: Microscope },
  { name: "dna", label: "DNA", keywords: "genetic sickle", icon: Dna },
  { name: "wind", label: "Wind", keywords: "respiratory lung asthma tb", icon: Wind },
  { name: "bean", label: "Bean", keywords: "kidney renal", icon: Bean },
  { name: "eye", label: "Eye", keywords: "ophthalmology vision", icon: Eye },
  { name: "ear", label: "Ear", keywords: "ent hearing", icon: Ear },
  { name: "hand", label: "Hand", keywords: "skin dermatology", icon: Hand },
  { name: "salad", label: "Salad", keywords: "nutrition diet malnutrition", icon: Salad },
  { name: "biohazard", label: "Biohazard", keywords: "outbreak ebola infection control virus viral covid", icon: Biohazard },
  { name: "bug", label: "Bug", keywords: "malaria mosquito vector", icon: Bug },
  { name: "worm", label: "Worm", keywords: "helminth deworming", icon: Worm },
  { name: "ribbon", label: "Ribbon", keywords: "hiv cancer", icon: Ribbon },
  { name: "skull", label: "Skull", keywords: "poison toxic", icon: Skull },
  { name: "shield-alert", label: "Shield alert", keywords: "safety risk", icon: ShieldAlert },
  { name: "shield-check", label: "Shield check", keywords: "safety prevention", icon: ShieldCheck },
  { name: "shield-plus", label: "Shield plus", keywords: "protection prevention", icon: ShieldPlus },
]

export const clinicalIconsByName = new Map(clinicalIcons.map((entry) => [entry.name, entry]))

export function isEmojiIcon(value: string): boolean {
  return /\p{Extended_Pictographic}/u.test(value)
}
