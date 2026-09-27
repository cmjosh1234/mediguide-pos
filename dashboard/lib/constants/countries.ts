import countries from "world-countries";

export type Country = {
  code: string;
  name: string;
};

/**
 * ISO 3166-1 countries by common English name, from the `world-countries`
 * package (bundled at build time, no runtime fetch). Guideline documents
 * store the name itself, matching existing rows and the public API's
 * case-insensitive `country` filter.
 */
export const COUNTRIES: Country[] = countries
  .map((country) => ({ code: country.cca2, name: country.name.common }))
  .sort((a, b) => a.name.localeCompare(b.name));
