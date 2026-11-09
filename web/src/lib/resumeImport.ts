export async function extractResume(file: File): Promise<string> {
  if (file.size > 10 * 1024 * 1024) throw new Error('Choose a resume smaller than 10 MB.')
  const extension = file.name.split('.').pop()?.toLowerCase()
  if (!['pdf', 'docx'].includes(extension ?? '')) throw new Error('Choose a PDF or DOCX resume.')
  const arrayBuffer = await file.arrayBuffer()
  let text = ''
  try {
    if (extension === 'docx') {
      const mammoth = await import('mammoth')
      text = (await mammoth.extractRawText({ arrayBuffer })).value
    } else {
      const pdfjs = await import('pdfjs-dist')
      const { default: worker } = await import('pdfjs-dist/build/pdf.worker.min.mjs?url')
      pdfjs.GlobalWorkerOptions.workerSrc = worker
      const loading = pdfjs.getDocument({ data: arrayBuffer })
      const document = await loading.promise
      try {
        if (document.numPages > 30) throw new Error('Resume is too long.')
        for (let i = 1; i <= document.numPages; i++) {
          const page = await document.getPage(i)
          const content = await page.getTextContent()
          text += content.items.map(item => 'str' in item ? item.str + (item.hasEOL ? '\n' : ' ') : '').join('') + '\n'
        }
      } finally { await loading.destroy() }
    }
  } catch {
    throw new Error('Could not read this file. It may be corrupt or password protected. Try another file or paste your resume text.')
  }
  if (!text.trim()) throw new Error('No readable text found. Scanned PDFs need OCR first, or you can paste your resume text below.')
  if (new TextEncoder().encode(text).length > 100 * 1024) throw new Error('Resume text must be smaller than 100 KB.')
  return text.trim()
}
