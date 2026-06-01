package dochtml

import "html/template"

var invoiceTmpl = template.Must(template.New("invoice").Funcs(funcMap).Parse(`<!DOCTYPE html>
<html lang="th">
<head>
<meta charset="UTF-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1"/>
<title>{{with index .Pages 0}}{{titleTH .Doc.Type}} — {{.Doc.DocumentNo}}{{end}}</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:'Sarabun','Tahoma',sans-serif;color:#000;font-size:10pt}
@media screen{
  body{background:#d0d0d0;padding:8mm 0}
  .page{width:210mm;min-height:297mm;background:#fff;margin:0 auto;padding:6mm 3mm;box-shadow:0 2px 12px rgba(0,0,0,0.2)}
}
@media print{body{background:#fff}.page{width:210mm;min-height:297mm;padding:6mm 3mm}@page{size:A4 portrait;margin:0}}
.hdr{display:flex;justify-content:space-between;align-items:flex-start;gap:8mm;margin-bottom:5mm}
.hdr>div:first-child{flex:1;min-width:0}
.store-name{font-size:14pt;font-weight:700;margin-bottom:1.5mm}
.meta-line{font-size:8.5pt;line-height:1.6}
.warn{font-style:italic}
.doc-side{text-align:right;flex-shrink:0}
.doc-sub{font-size:7.5pt;font-weight:600;letter-spacing:.5px;margin-bottom:1mm;border:1px solid #ccc;border-radius:4px;display:inline-block;padding:0.5mm 3mm;margin-bottom:2mm}
.doc-title{font-size:20pt;font-weight:800;margin-bottom:2.5mm}
.doc-info{font-size:8.5pt;border-collapse:collapse;margin-left:auto}
.doc-info td{padding:.8mm 2mm}
.doc-info .lbl{color:#444}
.doc-info .val{font-weight:600;font-family:monospace}
.rule{border:none;border-top:1.5px solid #ccc;margin-bottom:5mm}
.two-col{display:flex;gap:8mm;margin-bottom:6mm}
.col{flex:1}
.sec-lbl{font-size:7pt;text-transform:uppercase;letter-spacing:.5px;margin-bottom:1mm;color:#444;font-weight:600}
.cust-name{font-size:10.5pt;font-weight:700}
.cust-sub{font-size:8.5pt;margin-top:.5mm;line-height:1.5}
.mono{font-family:monospace}
.items{width:100%;border-collapse:collapse;margin-bottom:5mm;font-size:8.5pt;border-radius:6px;overflow:hidden}
.items th{padding:2mm 2.5mm;font-weight:700;background:#000;color:#fff;border:1px solid #ccc;text-align:center;line-height:1.3;print-color-adjust:exact;-webkit-print-color-adjust:exact}
.items th.l{text-align:left}
.items td{padding:2mm 2.5mm;border:.5px solid #ccc}
.r{text-align:right}.c{text-align:center}
.items tbody tr:last-child td{border-bottom:1px solid #ccc}
.tarea{display:flex;gap:8mm;margin-bottom:8mm;align-items:flex-start}
.notes{flex:1;min-width:0}
.notes-box{border:1px solid #ddd;border-radius:4px;padding:2.5mm 3mm;min-height:14mm}
.notes-lbl{font-weight:600;margin-bottom:1.5mm;font-size:7.5pt;color:#444}
.notes-content{font-size:8.5pt;color:#333;line-height:1.5}
.sumtbl{width:78mm;flex-shrink:0;font-size:8.5pt;border-collapse:collapse}
.sumtbl td{padding:1mm 2.5mm}
.sumtbl .slbl{color:#333}
.sumtbl .sv{text-align:right;font-family:monospace}
.total-row td{border-top:1.5px solid #ccc;font-size:11pt;font-weight:700;padding:1.5mm 2.5mm}
.sig-area{display:flex;gap:10mm;margin-top:10mm}
.sig-box{flex:1;text-align:center}
.sig-line{border-top:1px solid #ccc;padding-top:1.5mm;margin-top:14mm;font-size:8pt}
.sig-name{font-size:7.5pt;color:#333;margin-top:.5mm}
.footer{margin-top:5mm;border-top:.5px solid #888;padding-top:2mm;display:flex;justify-content:space-between;font-size:7pt;color:#444}
.page-break{break-after:page}
@media screen{.page-break{margin-bottom:12mm}}
.page-num{font-size:7.5pt;color:#555;text-align:right;margin-bottom:2mm}
.cont-hdr{display:flex;justify-content:space-between;align-items:center;border-bottom:1px solid #ccc;padding-bottom:2mm;margin-bottom:4mm}
.cont-hdr-name{font-weight:700;font-size:11pt}
.cont-hdr-ref{font-size:8.5pt;color:#444;text-align:right}
</style>
</head>
<body>
{{range .Pages}}
<div class="page{{if not .IsLast}} page-break{{end}}">

  {{/* Page number — top right */}}
  <div class="page-num">{{.PageNo}}/{{.TotalPages}}</div>

  {{if .IsFirst}}
  {{/* ── Full header (first page only) ── */}}
  <div class="hdr">
    <div style="display:flex;align-items:flex-start;gap:4mm">
      {{if .Store.LogoURL}}<img src="{{.Store.LogoURL}}" alt="logo" style="max-height:120px;width:auto;flex-shrink:0;display:block;margin-top:1mm">{{end}}
      <div>
        <div class="store-name">{{.Store.Name}}</div>
        {{if .Store.Address}}<div class="meta-line">{{.Store.Address}}</div>{{end}}
        {{if .Store.Phone}}<div class="meta-line">โทร: {{.Store.Phone}}</div>{{end}}
        {{if .Store.Fax}}<div class="meta-line">โทรสาร: {{.Store.Fax}}</div>{{end}}
        {{if or .Store.Email .Store.Website}}<div class="meta-line">{{if .Store.Email}}อีเมล: {{.Store.Email}}{{end}}{{if and .Store.Email .Store.Website}} · {{end}}{{if .Store.Website}}{{.Store.Website}}{{end}}</div>{{end}}
        <div class="meta-line">เลขประจำตัวผู้เสียภาษี: {{if .Store.TaxID}}<span class="mono">{{.Store.TaxID}}</span>{{else}}<span class="warn">ยังไม่ได้ตั้งค่า</span>{{end}}</div>
      </div>
    </div>
    <div class="doc-side">
      {{if isTaxDoc .Doc.Type}}<div class="doc-sub">ใบเสร็จรับเงิน / ใบกำกับภาษี</div>{{end}}
      <div class="doc-title">{{titleTH .Doc.Type}}</div>
      <table class="doc-info">
        <tr><td class="lbl">เลขที่</td><td class="val">{{.Doc.DocumentNo}}</td></tr>
        <tr><td class="lbl">เลขที่เต็ม</td><td class="val">{{.Doc.DocumentNoFull}}</td></tr>
        <tr><td class="lbl">วันที่</td><td>{{thaiDate .Doc.DocumentDate}}</td></tr>
        {{if .Doc.DueDate}}<tr><td class="lbl">กำหนดชำระ</td><td>{{derefTime .Doc.DueDate}}</td></tr>{{end}}
      </table>
    </div>
  </div>
  <hr class="rule"/>
  <div class="two-col">
    <div class="col">
      <div class="sec-lbl">ลูกค้า / Customer</div>
      <div class="cust-name">{{.Doc.CustomerName}}</div>
      {{if .Doc.CustomerAddress}}<div class="cust-sub">{{.Doc.CustomerAddress}}</div>{{end}}
      {{if .Doc.CustomerPhone}}<div class="cust-sub">โทร: {{.Doc.CustomerPhone}}</div>{{end}}
      {{if .Doc.CustomerTaxID}}<div class="cust-sub">เลขผู้เสียภาษี: <span class="mono">{{derefStr .Doc.CustomerTaxID}}</span></div>{{end}}
    </div>
    <div class="col">
      <div class="sec-lbl">ผู้ออกเอกสาร / Issued by</div>
      <div class="cust-name" style="font-size:9.5pt">{{.Doc.StaffName}}</div>
      <div class="cust-sub">{{titleEN .Doc.Type}}</div>
    </div>
  </div>
  {{else}}
  {{/* ── Compact header (continuation pages) ── */}}
  <div class="cont-hdr">
    <div class="cont-hdr-name">{{.Store.Name}}</div>
    <div class="cont-hdr-ref">{{titleTH .Doc.Type}} {{.Doc.DocumentNo}} (ต่อ)</div>
  </div>
  {{end}}

  {{/* ── Items table ── */}}
  <table class="items">
    <thead>
      <tr>
        <th class="c" style="width:8mm">#</th>
        <th class="l">รายการสินค้า / รายละเอียด<br>(PRODUCT DESCRIPTION)</th>
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
  {{/* ── Summary + signatures + footer (last page only) ── */}}
  <div class="tarea">
    <div class="notes">
      <div class="notes-lbl">หมายเหตุ</div>
      <div class="notes-box"><span class="notes-content">{{if .Doc.Notes}}{{derefStr .Doc.Notes}}{{end}}</span></div>
    </div>
    <table class="sumtbl">
      <tr><td class="slbl">ยอดรวมสินค้า</td><td class="sv">{{money .Doc.Subtotal}}</td></tr>
      {{if gt .Doc.TotalDiscount 0.0}}
      <tr style="border-top:.5px solid #ccc">
        <td class="slbl">ส่วนลดรวม</td>
        <td class="sv">-{{money .Doc.TotalDiscount}}</td>
      </tr>
      {{end}}
      {{if isTaxDoc .Doc.Type}}
      <tr style="border-top:.5px solid #ccc">
        <td class="slbl">ภาษีมูลค่าเพิ่ม {{fmtQty .Doc.VatRate}}% (แยกจากมูลค่า)</td>
        <td class="sv">{{money .Doc.VatAmount}}</td>
      </tr>
      {{else if gt .Doc.VatRate 0.0}}
      <tr style="border-top:.5px solid #ccc">
        <td class="slbl">ภาษีมูลค่าเพิ่ม {{fmtQty .Doc.VatRate}}%</td>
        <td class="sv">{{money .Doc.VatAmount}}</td>
      </tr>
      {{end}}
      <tr class="total-row">
        <td>รวมทั้งสิ้น</td>
        <td class="sv">{{money .Doc.TotalAmount}}</td>
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
    <span>ต้นฉบับ — {{titleTH .Doc.Type}} เลขที่ {{.Doc.DocumentNo}}</span>
    {{if .Store.TaxID}}<span>TIN: {{.Store.TaxID}}</span>{{end}}
    <span>{{thaiDate .Doc.DocumentDate}}</span>
  </div>
  {{end}}

</div>
{{end}}
</body>
</html>`))
