// Excel preview parser worker.
//
// SheetJS parsing of a multi-thousand-row workbook costs ~1s on the main
// thread — long enough to freeze the detail drawer while it opens. This
// module runs the parse off-thread: plain data in (ArrayBuffer or decoded
// text), plain AoA rows out. DOM mounting and pagination stay in
// document-preview.vue, which falls back to main-thread parsing when Workers
// are unavailable (and the encoding test runs without a DOM Worker).
import * as XLSX from 'xlsx';

export interface ExcelParseRequest {
  buffer?: ArrayBuffer;
  text?: string;
  fileType?: string;
}

export interface ExcelParsedSheet {
  name: string;
  rows: string[][];
  cols: number;
}

interface ExcelWorkerScope {
  onmessage: ((e: MessageEvent<ExcelParseRequest>) => void) | null;
  postMessage: (msg: { sheets?: ExcelParsedSheet[]; error?: string }) => void;
}

const scope = self as unknown as ExcelWorkerScope;

scope.onmessage = (e) => {
  try {
    const { buffer, text, fileType } = e.data || {};
    let workbook: XLSX.WorkBook;
    const lowerType = fileType?.toLowerCase();
    if (typeof text === 'string') {
      if (lowerType === 'tsv' || lowerType === 'tab') {
        workbook = XLSX.read(text, { type: 'string', FS: '\t' });
      } else {
        workbook = XLSX.read(text, { type: 'string' });
      }
    } else if (buffer) {
      workbook = XLSX.read(buffer, { type: 'array' });
    } else {
      throw new Error('empty payload');
    }
    // Read cell by cell inside the used range so absolute (1-based) row
    // numbers survive empty rows — citation locators address real rows.
    const sheets: ExcelParsedSheet[] = workbook.SheetNames.map((name) => {
      const ws = workbook.Sheets[name];
      const rows: string[][] = [];
      let cols = 0;
      const ref = ws?.['!ref'];
      if (ref) {
        const range = XLSX.utils.decode_range(ref);
        for (let r = range.s.r; r <= range.e.r; r++) {
          const cells: string[] = [];
          for (let c = range.s.c; c <= range.e.c; c++) {
            const cell = ws[XLSX.utils.encode_cell({ r, c })];
            cells.push(cell ? String(cell.w ?? cell.v ?? '') : '');
          }
          cols = Math.max(cols, cells.length);
          rows.push(cells);
        }
        rows.forEach((cells) => {
          while (cells.length < cols) cells.push('');
        });
      }
      return { name, rows, cols };
    });
    scope.postMessage({ sheets });
  } catch (err) {
    scope.postMessage({ error: err instanceof Error ? err.message : String(err) });
  }
};

export {};
