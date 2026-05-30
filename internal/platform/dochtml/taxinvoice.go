package dochtml

import (
	"bytes"
	"fmt"
	"html/template"
)

const (
	taxInvoiceItemsFirstPage = 25
	taxInvoiceItemsOtherPage = 26
)

var taxInvoiceTmpl = template.Must(template.New("taxInvoice").Funcs(template.FuncMap{
	"money":    formatMoney,
	"thaiDate": fmtThaiDate,
	"derefStr": derefStr,
	"inc":      func(i int) int { return i + 1 },
	"add":      func(a, b int) int { return a + b },
	"fmtQty":   func(f float64) string { return fmt.Sprintf("%.0f", f) },
}).Parse(`<!DOCTYPE html>
<html lang="th">
<head>
<meta charset="UTF-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1"/>
<title>{{with index .Pages 0}}ใบกำกับภาษี — {{.Doc.DocumentNo}}{{end}}</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:'Sarabun','Tahoma',sans-serif;color:#000;font-size:10pt}
@media screen{
  body{background:#d0d0d0;padding:8mm 0}
  .page{width:210mm;background:#fff;margin:0 auto 12mm;padding:8mm 6mm;box-shadow:0 2px 12px rgba(0,0,0,0.2)}
}
@media print{
  body{background:#fff;margin:0;padding:0}
  .page{width:auto;padding:0;box-shadow:none} /* margin handled by @page */
  .page-break{break-after:page}
  tr{break-inside:avoid}
  thead{display:table-header-group}
  .summary-section{break-inside:avoid}
  .sig-area{break-inside:avoid}
  .pg-badge{display:none!important}
  @page{size:A4 portrait;margin:8mm 6mm} /* paper margins here, not on .page */
}
/* ── Header ── */
.hdr{display:flex;justify-content:space-between;align-items:flex-start;gap:8mm;margin-bottom:4mm}
.hdr>div:first-child{flex:1;min-width:0}
.store-name{font-size:14pt;font-weight:700;margin-bottom:1.5mm}
.meta-line{font-size:8.5pt;line-height:1.7}
.warn{font-style:italic;color:#a00}
.mono{font-family:monospace}
/* ── Doc side ── */
.doc-side{text-align:right;flex-shrink:0;width:88mm}
.doc-title{font-size:22pt;font-weight:800;line-height:1.05}
.doc-title-en{font-size:9pt;font-weight:600;letter-spacing:.5px;margin-bottom:3mm}
.doc-info-wrap{border:1px solid #bbb;border-radius:6px;overflow:hidden}
.doc-info{width:100%;border-collapse:collapse;font-size:8.5pt}
.doc-info td{padding:1.6mm 2.5mm;border-bottom:.5px solid #ccc}
.doc-info tr:last-child td{border-bottom:none}
.doc-info .lbl{color:#222;text-align:left;width:60%;border-right:.5px solid #ccc}
.doc-info .val{font-weight:700;text-align:right}
/* ── Continuation header ── */
.cont-hdr{display:flex;justify-content:space-between;align-items:center;border-bottom:1.5px solid #000;padding-bottom:2mm;margin-bottom:4mm}
.cont-hdr-name{font-size:11pt;font-weight:700}
.cont-hdr-ref{font-size:8.5pt;color:#444}
/* ── Page number badge (screen only) ── */
.pg-badge{font-size:7.5pt;color:#555;text-align:right;margin-bottom:2mm}
.rule{border:none;border-top:1.5px solid #000;margin-bottom:4mm}
/* ── Buyer ── */
.buyer{margin-bottom:4mm}
.buyer-title{font-size:10pt;font-weight:700;margin-bottom:1.5mm}
.buyer-line{font-size:8.5pt;line-height:1.7}
.buyer-line .sep{color:#888;padding:0 2mm}
/* ── Items ── */
.items{width:100%;border-collapse:collapse;margin-bottom:4mm;font-size:8.5pt}
.items th{padding:2mm 2.5mm;font-weight:700;background:#1a1a1a;color:#fff;border:.5px solid #1a1a1a;text-align:center;line-height:1.3;-webkit-print-color-adjust:exact;print-color-adjust:exact}
.items td{padding:2mm 2.5mm;border:.5px solid #bbb}
.r{text-align:right}.c{text-align:center}
.items .desc-th{text-align:left}
.items .item-th{font-weight:700}
.items .item-en{color:#444;font-size:7.5pt}
/* ── Summary ── */
.tarea{display:flex;gap:6mm;align-items:stretch;margin-bottom:6mm}
.notes{flex:1;border:1px solid #ccc;border-radius:6px;padding:2.5mm 3mm}
.notes-lbl{font-weight:700;font-size:8pt;margin-bottom:2mm}
.notes-dots{font-size:8.5pt;color:#888;line-height:2.4}
.sumtbl{width:78mm;flex-shrink:0;font-size:8.5pt;border-collapse:collapse}
.sumtbl td{padding:1.8mm 2.5mm;border-bottom:.5px solid #ddd}
.slbl{color:#222}
.sv{text-align:right;font-family:monospace}
.total-row td{background:#f0f0f0;border-top:1.5px solid #000;border-bottom:none;font-size:10.5pt;font-weight:700;padding:2mm 2.5mm}
/* ── Signature ── */
.sig-area{display:flex;gap:8mm;margin-top:10mm}
.sig-box{flex:1;text-align:center;font-size:8.5pt;border:1px solid #bbb;border-radius:6px;padding:4mm}
.sig-role{font-weight:600;margin-bottom:12mm}
.sig-row{margin-bottom:4mm}
.sig-fill{display:inline-block;border-bottom:1px dotted #000;min-width:46mm}
/* ── Footer ── */
.footer{margin-top:5mm;border-top:.5px solid #888;padding-top:2mm;display:flex;justify-content:space-between;font-size:7pt;color:#444}
</style>
</head>
<body>
{{range .Pages}}
<div class="page{{if not .IsLast}} page-break{{end}}">

  {{/* Page number — screen only */}}
  <div class="pg-badge">หน้า {{.PageNo}} / {{.TotalPages}}</div>

  {{if .IsFirst}}
  {{/* ── Full header (first page) ── */}}
  <div class="hdr">
    <div>
      <div class="store-name">{{.Store.Name}}</div>
      {{if .Store.Address}}<div class="meta-line">{{.Store.Address}}</div>{{end}}
      {{if .Store.Phone}}<div class="meta-line">โทร. {{.Store.Phone}}</div>{{end}}
      <div class="meta-line">เลขประจำตัวผู้เสียภาษี {{if .Store.TaxID}}<span class="mono">{{.Store.TaxID}}</span>{{else}}<span class="warn">ยังไม่ได้ตั้งค่า</span>{{end}}</div>
      {{if .Store.Branch}}<div class="meta-line">สาขา {{.Store.Branch}}</div>{{end}}
    </div>
    <div class="doc-side">
      <div class="doc-title">ใบกำกับภาษี</div>
      <div class="doc-title-en">TAX INVOICE (FULL TAX INVOICE)</div>
      <div class="doc-info-wrap">
        <table class="doc-info">
          <tr><td class="lbl">เลขที่ใบกำกับภาษี (Tax Invoice No.)</td><td class="val">{{.Doc.DocumentNo}}</td></tr>
          <tr><td class="lbl">วันที่ออกใบกำกับภาษี (Tax Invoice Date)</td><td class="val">{{thaiDate .Doc.DocumentDate}}</td></tr>
          <tr><td class="lbl">เลขประจำตัวผู้เสียภาษี (ผู้ขาย)</td><td class="val">{{if .Store.TaxID}}{{.Store.TaxID}}{{else}}-{{end}}</td></tr>
        </table>
      </div>
    </div>
  </div>
  <hr class="rule"/>
  {{if .Doc.CustomerName}}
  <div class="buyer">
    <div class="buyer-title">ข้อมูลผู้ซื้อ (Buyer Information)</div>
    <div class="buyer-line">
      ชื่อบริษัท {{.Doc.CustomerName}}
      {{if .Doc.CustomerAddress}}<span class="sep">|</span>ที่อยู่ {{.Doc.CustomerAddress}}{{end}}
    </div>
    <div class="buyer-line">
      {{if .Doc.CustomerTaxID}}เลขประจำตัวผู้เสียภาษี <span class="mono">{{derefStr .Doc.CustomerTaxID}}</span>{{end}}
      {{if .Doc.CustomerBranch}}<span class="sep">|</span>สาขา {{derefStr .Doc.CustomerBranch}}{{end}}
      {{if .Doc.CustomerPhone}}<span class="sep">|</span>โทร. {{.Doc.CustomerPhone}}{{end}}
    </div>
  </div>
  {{end}}

  {{else}}
  {{/* ── Compact header (continuation pages) ── */}}
  <div class="cont-hdr">
    <div class="cont-hdr-name">{{.Store.Name}}</div>
    <div class="cont-hdr-ref">ใบกำกับภาษี {{.Doc.DocumentNo}} (ต่อ)</div>
  </div>
  {{end}}

  {{/* ── Items table ── */}}
  <table class="items">
    <thead>
      <tr>
        <th class="c" style="width:10mm">ลำดับ</th>
        <th class="c" style="width:24mm">รหัสสินค้า<br>(SKU)</th>
        <th class="desc-th">รายการสินค้า / รายละเอียด</th>
        <th class="c" style="width:16mm">จำนวน</th>
        <th class="c" style="width:14mm">หน่วย</th>
        <th class="c" style="width:26mm">ราคาต่อหน่วย<br>(บาท)</th>
        <th class="c" style="width:28mm">จำนวนเงิน<br>(บาท)</th>
      </tr>
    </thead>
    <tbody>
      {{$offset := .ItemOffset}}
      {{range $i, $item := .Items}}
      <tr>
        <td class="c">{{add $offset (inc $i)}}</td>
        <td class="c mono">{{if $item.SKU}}{{$item.SKU}}{{else}}-{{end}}</td>
        <td>
          <div class="item-th">{{$item.Description}}</div>
          {{if $item.DescriptionEn}}<div class="item-en">({{$item.DescriptionEn}})</div>{{end}}
        </td>
        <td class="c">{{fmtQty $item.Quantity}}</td>
        <td class="c">{{$item.Unit}}</td>
        <td class="r mono">{{money $item.UnitPrice}}</td>
        <td class="r mono" style="font-weight:600">{{money $item.Amount}}</td>
      </tr>
      {{end}}
      {{range .FillerRows}}<tr><td>&nbsp;</td><td></td><td></td><td></td><td></td><td></td><td></td></tr>{{end}}
    </tbody>
  </table>

  {{if .IsLast}}
  {{/* ── Summary + signatures + footer (last page only) ── */}}
  <div class="summary-section">
    <div class="tarea">
      <div class="notes">
        <div class="notes-lbl">หมายเหตุ (Remarks)</div>
        {{if .Doc.Notes}}
          <div style="font-size:8.5pt;color:#222">{{derefStr .Doc.Notes}}</div>
        {{else}}
          <div class="notes-dots">............................................................................<br>............................................................................</div>
        {{end}}
      </div>
      <table class="sumtbl">
        <tr><td class="slbl">รวมมูลค่าสินค้าก่อนภาษี (Subtotal)</td><td class="sv">{{money .Doc.Subtotal}}</td></tr>
        {{if gt .Doc.TotalDiscount 0.0}}
        <tr><td class="slbl">ส่วนลดรวม (Discount)</td><td class="sv">-{{money .Doc.TotalDiscount}}</td></tr>
        {{end}}
        {{if gt .Doc.VatRate 0.0}}
        <tr><td class="slbl">ภาษีมูลค่าเพิ่ม {{fmtQty .Doc.VatRate}}% (VAT {{fmtQty .Doc.VatRate}}%)</td><td class="sv">{{money .Doc.VatAmount}}</td></tr>
        {{end}}
        <tr class="total-row"><td>รวมทั้งสิ้น (Grand Total)</td><td class="sv">{{money .Doc.TotalAmount}}</td></tr>
      </table>
    </div>
    <div class="sig-area">
      <div class="sig-box">
        <div class="sig-role">ผู้ขาย / Seller</div>
        <div class="sig-row">ลงชื่อ <span class="sig-fill">&nbsp;</span></div>
        <div class="sig-row">( <span class="sig-fill">&nbsp;</span> )</div>
        <div class="sig-row">วันที่ <span class="sig-fill" style="min-width:14mm">&nbsp;</span> / <span class="sig-fill" style="min-width:14mm">&nbsp;</span> / <span class="sig-fill" style="min-width:14mm">&nbsp;</span></div>
      </div>
      <div class="sig-box">
        <div class="sig-role">ผู้รับสินค้า / Customer / Receiver</div>
        <div class="sig-row">ลงชื่อ <span class="sig-fill">&nbsp;</span></div>
        <div class="sig-row">( <span class="sig-fill">&nbsp;</span> )</div>
        <div class="sig-row">วันที่ <span class="sig-fill" style="min-width:14mm">&nbsp;</span> / <span class="sig-fill" style="min-width:14mm">&nbsp;</span> / <span class="sig-fill" style="min-width:14mm">&nbsp;</span></div>
      </div>
    </div>
    <div class="footer">
      <span>ต้นฉบับ — ใบกำกับภาษี เลขที่ {{.Doc.DocumentNo}}</span>
      {{if .Store.TaxID}}<span>TIN: {{.Store.TaxID}}</span>{{end}}
      <span>{{thaiDate .Doc.DocumentDate}}</span>
    </div>
  </div>
  {{end}}

</div>
{{end}}
</body>
</html>`))

func renderTaxInvoiceHTML(doc DocData, store StoreInfo) (string, error) {
	pages := paginateTaxInvoice(doc, store)
	var buf bytes.Buffer
	if err := taxInvoiceTmpl.Execute(&buf, renderData{Pages: pages}); err != nil {
		return "", fmt.Errorf("tax invoice html: %w", err)
	}
	return buf.String(), nil
}

func paginateTaxInvoice(doc DocData, store StoreInfo) []pageData {
	return paginateWithLimits(doc, store, taxInvoiceItemsFirstPage, taxInvoiceItemsOtherPage)
}
