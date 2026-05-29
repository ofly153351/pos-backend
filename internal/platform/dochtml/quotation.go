package dochtml

import (
	"bytes"
	"fmt"
	"html/template"
)

var quotationTmpl = template.Must(template.New("quotation").Funcs(template.FuncMap{
	"money":     formatMoney,
	"thaiDate":  fmtThaiDate,
	"derefStr":  derefStr,
	"derefTime": derefTime,
	"isTaxDoc":  isTaxDoc,
	"inc":       func(i int) int { return i + 1 },
	"add":       func(a, b int) int { return a + b },
	"fmtQty":    func(f float64) string { return fmt.Sprintf("%.0f", f) },
}).Parse(`<!DOCTYPE html>
<html lang="th">
<head>
<meta charset="UTF-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1"/>
<title>{{with index .Pages 0}}ใบเสนอราคา — {{.Doc.DocumentNo}}{{end}}</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:'Sarabun','Tahoma',sans-serif;color:#000;font-size:10pt}
@media screen{
  body{background:#d0d0d0;padding:8mm 0}
  .page{width:210mm;min-height:297mm;background:#fff;margin:0 auto;padding:6mm 3mm;box-shadow:0 2px 12px rgba(0,0,0,0.2)}
}
@media print{body{background:#fff}.page{width:210mm;min-height:297mm;padding:6mm 3mm}@page{size:A4 portrait;margin:0}}
/* ── Header ── */
.hdr{display:flex;justify-content:space-between;align-items:flex-start;gap:8mm;margin-bottom:5mm}
.hdr>div:first-child{flex:1;min-width:0}
.store-name{font-size:14pt;font-weight:700;margin-bottom:1.5mm}
.meta-line{font-size:8.5pt;line-height:1.6}
.warn{font-style:italic}
/* ── Doc side ── */
.doc-side{text-align:right;flex-shrink:0}
.doc-badge{display:inline-block;border:1.5px solid #000;padding:1mm 5mm;font-size:8.5pt;font-weight:700;letter-spacing:.5px;margin-bottom:2mm}
.doc-title{font-size:22pt;font-weight:800;margin-bottom:2.5mm}
.doc-info{font-size:8.5pt;border-collapse:collapse;margin-left:auto}
.doc-info td{padding:.8mm 2mm}
.doc-info .lbl{color:#444}
.doc-info .val{font-weight:600;font-family:monospace}
.doc-info .valid{color:#000;font-weight:700}
.rule{border:none;border-top:1.5px solid #000;margin-bottom:5mm}
.mono{font-family:monospace}
/* ── Customer ── */
.cust-section{margin-bottom:5mm;padding:3mm 4mm;border:1px solid #ccc;border-radius:3px}
.cust-lbl{font-size:7.5pt;text-transform:uppercase;letter-spacing:.5px;color:#444;font-weight:700;margin-bottom:1mm}
.cust-name{font-size:11pt;font-weight:700}
.cust-sub{font-size:8.5pt;margin-top:.5mm;line-height:1.5}
/* ── Items ── */
.items{width:100%;border-collapse:collapse;margin-bottom:5mm;font-size:8.5pt}
.items th{padding:1.5mm 2.5mm;font-weight:700;border-top:1.5px solid #000;border-bottom:1px solid #000;text-align:left}
.items td{padding:1.5mm 2.5mm;border-bottom:.5px solid #ccc}
.r{text-align:right}.c{text-align:center}
.items tbody tr:last-child td{border-bottom:1px solid #000}
/* ── Summary ── */
.tarea{display:flex;gap:8mm;align-items:flex-start;margin-bottom:8mm}
.notes{flex:1}
.notes-lbl{font-weight:600;font-size:7.5pt;color:#444;margin-bottom:1.5mm}
.notes-box{border:1px solid #ddd;border-radius:4px;padding:2.5mm 3mm;min-height:14mm;font-size:8.5pt}
.sumtbl{width:78mm;flex-shrink:0;font-size:8.5pt;border-collapse:collapse}
.sumtbl td{padding:1mm 2.5mm}
.slbl{color:#333}
.sv{text-align:right;font-family:monospace}
.total-row td{border-top:1.5px solid #000;font-size:11pt;font-weight:700;padding:1.5mm 2.5mm}
/* ── Validity notice ── */
.validity-box{border:1px dashed #888;padding:2mm 4mm;margin-bottom:5mm;font-size:8.5pt;color:#333}
/* ── Signature ── */
.sig-area{display:flex;gap:10mm;margin-top:8mm}
.sig-box{flex:1;text-align:center}
.sig-line{border-top:1px solid #000;padding-top:1.5mm;margin-top:14mm;font-size:8pt}
.sig-name{font-size:7.5pt;color:#333;margin-top:.5mm}
.stamp-box{flex:1;text-align:center;border:1px dashed #bbb;height:30mm;display:flex;align-items:center;justify-content:center;font-size:8pt;color:#999}
/* ── Footer ── */
.footer{margin-top:5mm;border-top:.5px solid #888;padding-top:2mm;display:flex;justify-content:space-between;font-size:7pt;color:#444}
</style>
</head>
<body>
{{range .Pages}}
<div class="page{{if not .IsLast}} page-break{{end}}">
  <div class="hdr">
    <div>
      {{if .Store.LogoURL}}<div style="margin-bottom:3mm"><img src="{{.Store.LogoURL}}" alt="logo" style="max-height:120px;width:auto;display:block"></div>{{end}}
      <div class="store-name">{{.Store.Name}}</div>
      {{if .Store.Address}}<div class="meta-line">{{.Store.Address}}</div>{{end}}
      {{if .Store.Phone}}<div class="meta-line">โทร: {{.Store.Phone}}</div>{{end}}
      {{if .Store.Fax}}<div class="meta-line">โทรสาร: {{.Store.Fax}}</div>{{end}}
      {{if or .Store.Email .Store.Website}}<div class="meta-line">{{if .Store.Email}}อีเมล: {{.Store.Email}}{{end}}{{if and .Store.Email .Store.Website}} · {{end}}{{if .Store.Website}}{{.Store.Website}}{{end}}</div>{{end}}
      <div class="meta-line">เลขประจำตัวผู้เสียภาษี: {{if .Store.TaxID}}<span class="mono">{{.Store.TaxID}}</span>{{else}}<span class="warn">ยังไม่ได้ตั้งค่า</span>{{end}}</div>
    </div>
    <div class="doc-side">
      <div class="doc-badge">QUOTATION</div>
      <div class="doc-title">ใบเสนอราคา</div>
      <table class="doc-info">
        <tr><td class="lbl">เลขที่</td><td class="val">{{.Doc.DocumentNo}}</td></tr>
        <tr><td class="lbl">เลขที่เต็ม</td><td class="val">{{.Doc.DocumentNoFull}}</td></tr>
        <tr><td class="lbl">วันที่</td><td>{{thaiDate .Doc.DocumentDate}}</td></tr>
        {{if .Doc.ValidUntil}}<tr><td class="lbl">ยืนราคาถึง</td><td class="valid">{{derefTime .Doc.ValidUntil}}</td></tr>{{end}}
      </table>
    </div>
  </div>

  <hr class="rule"/>

  {{if .Doc.CustomerName}}
  <div class="cust-section">
    <div class="cust-lbl">เรียน / Attention</div>
    <div class="cust-name">{{.Doc.CustomerName}}</div>
    {{if .Doc.CustomerAddress}}<div class="cust-sub">{{.Doc.CustomerAddress}}</div>{{end}}
    {{if .Doc.CustomerPhone}}<div class="cust-sub">โทร: {{.Doc.CustomerPhone}}</div>{{end}}
    {{if .Doc.CustomerTaxID}}<div class="cust-sub">เลขผู้เสียภาษี: <span class="mono">{{derefStr .Doc.CustomerTaxID}}</span></div>{{end}}
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
  <div class="tarea">
    <div class="notes">
      <div class="notes-lbl">หมายเหตุ</div>
      <div class="notes-box"><span style="font-size:8.5pt;color:#333">{{if .Doc.Notes}}{{derefStr .Doc.Notes}}{{end}}</span></div>
    </div>
    <table class="sumtbl">
      <tr><td class="slbl">ยอดรวมสินค้า</td><td class="sv">{{money .Doc.Subtotal}}</td></tr>
      {{if gt .Doc.TotalDiscount 0.0}}
      <tr style="border-top:.5px solid #ccc"><td class="slbl">ส่วนลดรวม</td><td class="sv">-{{money .Doc.TotalDiscount}}</td></tr>
      {{end}}
      {{if gt .Doc.VatRate 0.0}}
      <tr style="border-top:.5px solid #ccc">
        <td class="slbl">ภาษีมูลค่าเพิ่ม {{fmtQty .Doc.VatRate}}%</td>
        <td class="sv">{{money .Doc.VatAmount}}</td>
      </tr>
      {{end}}
      <tr class="total-row">
        <td>ยอดรวมสุทธิ</td>
        <td class="sv">{{money .Doc.TotalAmount}}</td>
      </tr>
    </table>
  </div>

  <div class="validity-box">
    {{if .Doc.ValidUntil}}ราคานี้ยืนราคาถึงวันที่ {{derefTime .Doc.ValidUntil}} · {{else}}ราคานี้ยืนราคา 30 วันนับจากวันที่ออกเอกสาร · {{end}}ราคาอาจเปลี่ยนแปลงได้โดยไม่แจ้งล่วงหน้า
  </div>

  <div class="sig-area">
    <div class="sig-box">
      <div class="sig-line">ผู้เสนอราคา / Authorized Signature</div>
      <div class="sig-name">{{.Store.Name}}</div>
    </div>
    <div class="stamp-box">ตราประทับ / Company Seal</div>
    <div class="sig-box">
      <div class="sig-line">ผู้อนุมัติ / Approved by</div>
      <div class="sig-name">&nbsp;</div>
    </div>
  </div>

  <div class="footer">
    <span>ต้นฉบับ — ใบเสนอราคา เลขที่ {{.Doc.DocumentNo}}</span>
    {{if .Store.TaxID}}<span>TIN: {{.Store.TaxID}}</span>{{end}}
    <span>{{thaiDate .Doc.DocumentDate}}</span>
  </div>
  {{end}}
</div>
{{end}}
</body>
</html>`))

func renderQuotationHTML(doc DocData, store StoreInfo) (string, error) {
	pages := paginateQuotation(doc, store)
	var buf bytes.Buffer
	if err := quotationTmpl.Execute(&buf, renderData{Pages: pages}); err != nil {
		return "", fmt.Errorf("quotation html: %w", err)
	}
	return buf.String(), nil
}

func paginateQuotation(doc DocData, store StoreInfo) []pageData {
	// Same pagination logic as invoice
	return paginateWithLimits(doc, store, itemsPerFirstPage, itemsPerOtherPage)
}
