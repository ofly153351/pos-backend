package dochtml

import (
	"bytes"
	"fmt"
	"html/template"
)

var taxInvoiceTmpl = template.Must(template.New("taxinvoice").Funcs(template.FuncMap{
	"money":    formatMoney,
	"thaiDate": fmtThaiDate,
	"derefStr": derefStr,
	"derefTime": derefTime,
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
  .page{width:210mm;min-height:297mm;background:#fff;margin:0 auto;padding:6mm 3mm;box-shadow:0 2px 12px rgba(0,0,0,0.2)}
}
@media print{body{background:#fff}.page{width:210mm;min-height:297mm;padding:6mm 3mm}@page{size:A4 portrait;margin:0}}
.page-break{break-after:page}
@media screen{.page-break{margin-bottom:12mm}}
/* ── Header ── */
.hdr{display:flex;justify-content:space-between;align-items:flex-start;gap:8mm;margin-bottom:5mm}
.hdr>div:first-child{flex:1;min-width:0}
.store-name{font-size:14pt;font-weight:700;margin-bottom:1.5mm}
.meta-line{font-size:8.5pt;line-height:1.6}
.tin-seller{font-size:9pt;font-weight:700;margin-top:1.5mm;padding:1mm 2mm;border:1px solid #000;display:inline-block}
.warn{font-style:italic;color:#b45309}
/* ── Doc side ── */
.doc-side{text-align:right;flex-shrink:0}
.doc-label{font-size:9pt;color:#444;font-weight:600;letter-spacing:.5px;margin-bottom:.5mm}
.doc-title{font-size:22pt;font-weight:900;margin-bottom:1mm;letter-spacing:-0.5px}
.doc-subtitle{font-size:9pt;color:#444;margin-bottom:3mm}
.doc-info{font-size:8.5pt;border-collapse:collapse;margin-left:auto}
.doc-info td{padding:.8mm 2mm}
.doc-info .lbl{color:#444}
.doc-info .val{font-weight:700;font-family:monospace}
/* ── Rule ── */
.rule{border:none;border-top:2px solid #000;margin-bottom:5mm}
.rule-thin{border:none;border-top:.5px solid #888;margin-bottom:4mm}
.mono{font-family:monospace}
/* ── Buyer box ── */
.buyer-box{border:1px solid #000;padding:3mm 4mm;margin-bottom:5mm}
.buyer-lbl{font-size:7.5pt;text-transform:uppercase;letter-spacing:.5px;font-weight:700;color:#444;margin-bottom:1mm}
.buyer-name{font-size:11pt;font-weight:700}
.buyer-sub{font-size:8.5pt;margin-top:.5mm;line-height:1.5}
.tin-buyer{font-size:9pt;font-weight:700;margin-top:1mm}
.tin-required{color:#b45309;font-style:italic;font-size:8pt}
/* ── Items ── */
.items{width:100%;border-collapse:collapse;margin-bottom:5mm;font-size:8.5pt}
.items th{padding:1.5mm 2.5mm;font-weight:700;border-top:2px solid #000;border-bottom:1px solid #000;text-align:left;background:#f8f8f8}
.items td{padding:1.5mm 2.5mm;border-bottom:.5px solid #ccc}
.r{text-align:right}.c{text-align:center}
.items tbody tr:last-child td{border-bottom:1.5px solid #000}
/* ── Summary — Tax focused ── */
.summary-wrap{display:flex;gap:8mm;align-items:flex-start;margin-bottom:8mm}
.notes{flex:1}
.notes-lbl{font-weight:600;font-size:7.5pt;color:#444;margin-bottom:1.5mm}
.notes-box{border:1px solid #ddd;border-radius:4px;padding:2.5mm 3mm;min-height:14mm;font-size:8.5pt}
.tax-summary{width:82mm;flex-shrink:0;border:1px solid #000;font-size:8.5pt}
.tax-summary tr td{padding:1.5mm 3mm}
.tax-summary .lbl{color:#333;border-bottom:.5px solid #e5e5e5}
.tax-summary .val{text-align:right;font-family:monospace;border-bottom:.5px solid #e5e5e5}
.tax-summary .vat-row td{background:#f5f5f5;font-weight:600;border-top:1px solid #888;border-bottom:1px solid #888}
.tax-summary .total-row td{font-size:11pt;font-weight:800;border-top:2px solid #000;padding:2mm 3mm}
/* ── Signature ── */
.sig-area{display:flex;gap:10mm;margin-top:8mm}
.sig-box{flex:1;text-align:center}
.sig-line{border-top:1px solid #000;padding-top:1.5mm;margin-top:14mm;font-size:8pt}
.sig-name{font-size:7.5pt;color:#333;margin-top:.5mm}
/* ── Footer ── */
.footer{margin-top:5mm;border-top:1px solid #000;padding-top:2mm;display:flex;justify-content:space-between;align-items:center;font-size:7.5pt;color:#444}
.copy-mark{font-size:9pt;font-weight:700;border:1.5px solid #000;padding:.5mm 4mm}
/* ── Page num ── */
.page-num{font-size:7.5pt;color:#555;text-align:right;margin-bottom:2mm}
.cont-hdr{display:flex;justify-content:space-between;align-items:center;border-bottom:1.5px solid #000;padding-bottom:2mm;margin-bottom:4mm}
.cont-hdr-name{font-weight:700;font-size:11pt}
.cont-hdr-ref{font-size:8.5pt;color:#444;text-align:right}
</style>
</head>
<body>
{{range .Pages}}
<div class="page{{if not .IsLast}} page-break{{end}}">

  <div class="page-num">{{.PageNo}}/{{.TotalPages}}</div>

  {{if .IsFirst}}
  <div class="hdr">
    <div>
      {{if .Store.LogoURL}}<div style="margin-bottom:3mm"><img src="{{.Store.LogoURL}}" alt="logo" style="max-height:120px;width:auto;display:block"></div>{{end}}
      <div class="store-name">{{.Store.Name}}</div>
      {{if .Store.Address}}<div class="meta-line">{{.Store.Address}}</div>{{end}}
      {{if .Store.Phone}}<div class="meta-line">โทร: {{.Store.Phone}}</div>{{end}}
      {{if .Store.Fax}}<div class="meta-line">โทรสาร: {{.Store.Fax}}</div>{{end}}
      {{if or .Store.Email .Store.Website}}<div class="meta-line">{{if .Store.Email}}อีเมล: {{.Store.Email}}{{end}}{{if and .Store.Email .Store.Website}} · {{end}}{{if .Store.Website}}{{.Store.Website}}{{end}}</div>{{end}}
      {{if .Store.TaxID}}
        <div class="tin-seller">เลขประจำตัวผู้เสียภาษี: <span class="mono">{{.Store.TaxID}}</span></div>
      {{else}}
        <div class="tin-seller warn">เลขประจำตัวผู้เสียภาษี: ยังไม่ได้ตั้งค่า ⚠</div>
      {{end}}
    </div>
    <div class="doc-side">
      <div class="doc-label">ต้นฉบับ / ORIGINAL</div>
      <div class="doc-title">ใบกำกับภาษี</div>
      <div class="doc-subtitle">Tax Invoice</div>
      <table class="doc-info">
        <tr><td class="lbl">เลขที่</td><td class="val">{{.Doc.DocumentNo}}</td></tr>
        <tr><td class="lbl">เลขที่เต็ม</td><td class="val">{{.Doc.DocumentNoFull}}</td></tr>
        <tr><td class="lbl">วันที่</td><td>{{thaiDate .Doc.DocumentDate}}</td></tr>
        {{if .Doc.DueDate}}<tr><td class="lbl">กำหนดชำระ</td><td>{{derefTime .Doc.DueDate}}</td></tr>{{end}}
      </table>
    </div>
  </div>

  <hr class="rule"/>

  <div class="buyer-box">
    <div class="buyer-lbl">ผู้ซื้อ / Buyer</div>
    <div class="buyer-name">{{.Doc.CustomerName}}</div>
    {{if .Doc.CustomerAddress}}<div class="buyer-sub">{{.Doc.CustomerAddress}}</div>{{end}}
    {{if .Doc.CustomerPhone}}<div class="buyer-sub">โทร: {{.Doc.CustomerPhone}}</div>{{end}}
    <div class="tin-buyer">เลขประจำตัวผู้เสียภาษี: {{if .Doc.CustomerTaxID}}<span class="mono">{{derefStr .Doc.CustomerTaxID}}</span>{{else}}<span class="tin-required">ยังไม่ได้ระบุ</span>{{end}}</div>
  </div>
  {{else}}
  <div class="cont-hdr">
    <div class="cont-hdr-name">{{.Store.Name}}</div>
    <div class="cont-hdr-ref">ใบกำกับภาษี {{.Doc.DocumentNo}} (ต่อ)</div>
  </div>
  {{end}}

  <table class="items">
    <thead>
      <tr>
        <th class="c" style="width:8mm">#</th>
        <th>รายการสินค้า / บริการ</th>
        <th class="c" style="width:14mm">หน่วย</th>
        <th class="c" style="width:14mm">จำนวน</th>
        <th class="r" style="width:26mm">ราคา/หน่วย</th>
        <th class="r" style="width:22mm">ส่วนลด</th>
        <th class="r" style="width:28mm">จำนวนเงิน</th>
      </tr>
    </thead>
    <tbody>
      {{$offset := .ItemOffset}}
      {{range $i, $item := .Items}}
      <tr>
        <td class="c">{{add $offset (inc $i)}}</td>
        <td>{{$item.Description}}</td>
        <td class="c">{{$item.Unit}}</td>
        <td class="c">{{fmtQty $item.Quantity}}</td>
        <td class="r mono">{{money $item.UnitPrice}}</td>
        <td class="r mono">{{if gt $item.DiscountValue 0.0}}-{{money $item.DiscountValue}}{{else}}0{{end}}</td>
        <td class="r mono" style="font-weight:600">{{money $item.Amount}}</td>
      </tr>
      {{end}}
      {{range .FillerRows}}<tr><td>&nbsp;</td><td></td><td></td><td></td><td></td><td></td><td></td></tr>{{end}}
    </tbody>
  </table>

  {{if .IsLast}}
  <div class="summary-wrap">
    <div class="notes">
      <div class="notes-lbl">หมายเหตุ</div>
      <div class="notes-box"><span style="font-size:8.5pt;color:#333">{{if .Doc.Notes}}{{derefStr .Doc.Notes}}{{end}}</span></div>
    </div>
    <table class="tax-summary">
      <tr>
        <td class="lbl">มูลค่าสินค้า/บริการ</td>
        <td class="val">{{money .Doc.Subtotal}}</td>
      </tr>
      {{if gt .Doc.TotalDiscount 0.0}}
      <tr>
        <td class="lbl">ส่วนลดรวม</td>
        <td class="val">-{{money .Doc.TotalDiscount}}</td>
      </tr>
      {{end}}
      <tr class="vat-row">
        <td>ภาษีมูลค่าเพิ่ม {{fmtQty .Doc.VatRate}}% <span style="font-size:7.5pt;font-weight:400">(แยกต่างหาก)</span></td>
        <td class="val">{{money .Doc.VatAmount}}</td>
      </tr>
      <tr class="total-row">
        <td>ราคารวมภาษีมูลค่าเพิ่ม</td>
        <td class="val">{{money .Doc.TotalAmount}}</td>
      </tr>
    </table>
  </div>

  <div class="sig-area">
    <div class="sig-box">
      <div class="sig-line">ผู้รับเงิน / ผู้มีอำนาจลงนาม</div>
      <div class="sig-name">{{.Store.Name}}</div>
    </div>
    <div class="sig-box">
      <div class="sig-line">ผู้จ่ายเงิน / ลูกค้า</div>
      <div class="sig-name">{{.Doc.CustomerName}}</div>
    </div>
  </div>

  <div class="footer">
    <span class="copy-mark">ต้นฉบับ</span>
    <span style="font-size:7pt">ใบกำกับภาษี เลขที่ {{.Doc.DocumentNo}} · {{thaiDate .Doc.DocumentDate}}</span>
    {{if .Store.TaxID}}<span style="font-size:7pt">TIN: {{.Store.TaxID}}</span>{{end}}
  </div>
  {{end}}

</div>
{{end}}
</body>
</html>`))

func renderTaxInvoiceHTML(doc DocData, store StoreInfo) (string, error) {
	pages := paginateWithLimits(doc, store, itemsPerFirstPage, itemsPerOtherPage)
	var buf bytes.Buffer
	if err := taxInvoiceTmpl.Execute(&buf, renderData{Pages: pages}); err != nil {
		return "", fmt.Errorf("tax invoice html: %w", err)
	}
	return buf.String(), nil
}
