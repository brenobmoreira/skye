import { dropSpot, type RowBox, type Spot } from '../lib/flip';

// spotUnder says where a dragged item lands in a list for a pointer at clientY, among the
// [data-flip] rows that ok lets it go among. box must be the rows' offsetParent (positioned) and
// the list that scrolls: the rows are measured where layout puts them, not where a slide draws
// them, so a row sliding under the pointer does not decide.
export function spotUnder(box: HTMLElement, clientY: number, ok: (id: string) => boolean): Spot | null {
  const y = clientY - box.getBoundingClientRect().top - box.clientTop + box.scrollTop;
  const rows: RowBox[] = [];
  box.querySelectorAll<HTMLElement>('[data-flip]').forEach((n) => {
    const id = n.dataset.flip ?? '';
    if (ok(id)) rows.push({ id, top: n.offsetTop, height: n.offsetHeight });
  });
  return dropSpot(rows, y);
}
