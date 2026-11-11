// capitalize raises the first letter, for a label that stands alone at the start of a line.
export function capitalize(s: string): string {
  return s.charAt(0).toLocaleUpperCase('pt-BR') + s.slice(1);
}
