package dochtml

import "html/template"

var billTmpl = template.Must(template.New("bill").Funcs(funcMap).Parse(`<!DOCTYPE html>
<html lang="th">
<head>
<meta charset="UTF-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1"/>
<title>ใบวางบิล — {{.Doc.DocumentNo}}</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:'Sarabun','Tahoma',sans-serif;color:#000;background:#fff;font-size:10pt}
@media screen{.page{width:210mm;min-height:297mm;background:#fff;margin:0 auto;padding:6mm 3mm}}
@media print{.page{width:210mm;min-height:297mm;padding:6mm 3mm}@page{size:A4 portrait;margin:0}}
.hdr{display:flex;justify-content:space-between;align-items:flex-start;gap:8mm;margin-bottom:5mm}
.hdr>div:first-child{flex:1;min-width:0}
.store-name{font-size:14pt;font-weight:700;margin-bottom:1.5mm}
.meta-line{font-size:8.5pt;line-height:1.6}
.warn{font-style:italic}
.doc-side{text-align:right;flex-shrink:0}
.doc-title{font-size:18pt;font-weight:800;margin-bottom:2mm}
.doc-sub{font-size:7.5pt;font-weight:600;border:1px solid #000;display:inline-block;padding:.5mm 3mm;margin-bottom:2mm;letter-spacing:.5px}
.doc-meta{font-size:8.5pt;margin-top:1mm}
.mono{font-family:monospace;font-weight:600}
.rule{border:none;border-top:1.5px solid #000;margin:4mm 0}
.pay-box{border:1.5px solid #000;padding:4mm 6mm;margin-bottom:6mm;display:flex;justify-content:space-between;align-items:center}
.pay-left .lbl{font-size:7.5pt;color:#444;margin-bottom:1mm}
.pay-left .due{font-size:12pt;font-weight:700}
.pay-right{text-align:right}
.pay-right .lbl{font-size:7.5pt;color:#444;margin-bottom:1mm}
.pay-right .amount{font-size:20pt;font-weight:800;font-family:monospace}
.two-col{display:flex;gap:8mm;margin-bottom:5mm}
.col{flex:1}
.sec-lbl{font-size:7pt;text-transform:uppercase;letter-spacing:.5px;font-weight:700;margin-bottom:1mm;color:#444}
.cust-name{font-size:10.5pt;font-weight:700}
.cust-sub{font-size:8.5pt;margin-top:.5mm;line-height:1.5}
.items{width:100%;border-collapse:collapse;margin-bottom:5mm;font-size:8.5pt}
.items th{padding:1.5mm 2.5mm;font-weight:700;border-top:1.5px solid #000;border-bottom:1px solid #000}
.items td{padding:1.5mm 2.5mm;border-bottom:.5px solid #ccc}
.items tbody tr:last-child td{border-bottom:1px solid #000}
.r{text-align:right}.c{text-align:center}
.totals{display:flex;justify-content:flex-end;margin-bottom:6mm}
.sum-tbl{min-width:78mm;font-size:8.5pt;border-collapse:collapse}
.sum-tbl td{padding:1mm 2.5mm}
.slbl{color:#333}
.sv{text-align:right;font-family:monospace}
.total-row td{border-top:1.5px solid #000;font-size:11pt;font-weight:700;padding:1.5mm 2.5mm}
.notes{margin-bottom:5mm}
.notes-box{border:1px solid #ddd;border-radius:4px;padding:2.5mm 3mm;min-height:14mm}
.notes-lbl{font-weight:700;font-size:7.5pt;color:#444;margin-bottom:1.5mm}
.notes-content{font-size:8.5pt;color:#333;line-height:1.5}
.footer{border-top:.5px solid #888;padding-top:2mm;display:flex;justify-content:space-between;font-size:7pt;color:#444;margin-top:4mm}
</style>
</head>
<body>
<div class="page">

  <div class="hdr">
    <div style="display:flex;align-items:flex-start;gap:4mm">
      {{if .Store.LogoURL}}<img src="{{.Store.LogoURL}}" alt="logo" style="max-height:120px;width:auto;flex-shrink:0;display:block;margin-top:1mm">{{end}}
      <div>
        <div class="store-name">{{.Store.Name}}</div>
        {{if .Store.Address}}<div class="meta-line">{{.Store.Address}}</div>{{end}}
        {{if .Store.Phone}}<div class="meta-line">โทร: {{.Store.Phone}}</div>{{end}}{{if .Store.Fax}}<div class="meta-line">โทรสาร: {{.Store.Fax}}</div>{{end}}
        {{if .Store.Email}}<div class="meta-line">อีเมล: {{.Store.Email}}</div>{{end}}{{if .Store.Website}}<div class="meta-line">{{.Store.Website}}</div>{{end}}
        <div class="meta-line">เลขประจำตัวผู้เสียภาษี: {{if .Store.TaxID}}<span class="mono">{{.Store.TaxID}}</span>{{else}}<span class="warn">ยังไม่ได้ตั้งค่า</span>{{end}}</div>
      </div>
    </div>
    <div class="doc-side">
      <div class="doc-sub">BILLING NOTICE</div>
      <div class="doc-title">ใบวางบิล</div>
      <div class="doc-meta">เลขที่ <span class="mono">{{.Doc.DocumentNo}}</span></div>
      <div class="doc-meta">วันที่ {{thaiDate .Doc.DocumentDate}}</div>
    </div>
  </div>

  <hr class="rule"/>

  <div class="pay-box">
    <div class="pay-left">
      <div class="lbl">กรุณาชำระเงินภายใน / Please Pay By</div>
      {{if .Doc.DueDate}}
      <div class="due">{{derefTime .Doc.DueDate}}</div>
      {{else}}
      <div class="due">ไม่ระบุวันกำหนด</div>
      {{end}}
    </div>
    <div class="pay-right">
      <div class="lbl">ยอดที่ต้องชำระ / Amount Due</div>
      <div class="amount">฿{{money .Doc.TotalAmount}}</div>
    </div>
  </div>

  <hr class="rule"/>

  <div class="two-col">
    <div class="col">
      <div class="sec-lbl">เรียน / Billed To</div>
      <div class="cust-name">{{.Doc.CustomerName}}</div>
      {{if .Doc.CustomerAddress}}<div class="cust-sub">{{.Doc.CustomerAddress}}</div>{{end}}
      {{if .Doc.CustomerPhone}}<div class="cust-sub">โทร: {{.Doc.CustomerPhone}}</div>{{end}}
      {{if .Doc.CustomerTaxID}}<div class="cust-sub">เลขผู้เสียภาษี: <span class="mono">{{derefStr .Doc.CustomerTaxID}}</span></div>{{end}}
    </div>
    <div class="col">
      <div class="sec-lbl">ออกโดย / Issued By</div>
      <div class="cust-name" style="font-size:9.5pt">{{.Store.Name}}</div>
      {{if .Store.Address}}<div class="cust-sub">{{.Store.Address}}</div>{{end}}
      <div class="cust-sub">ผู้ออกเอกสาร: {{.Doc.StaffName}}</div>
    </div>
  </div>

  <table class="items">
    <thead>
      <tr>
        <th class="c" style="width:8mm">#</th>
        <th>รายการ / Description</th>
        <th class="c" style="width:14mm">หน่วย</th>
        <th class="c" style="width:16mm">จำนวน</th>
        <th class="r" style="width:30mm">ราคา/หน่วย</th>
        <th class="r" style="width:30mm">จำนวนเงิน</th>
      </tr>
    </thead>
    <tbody>
      {{range $i, $item := .Doc.Items}}
      <tr>
        <td class="c">{{inc $i}}</td>
        <td>{{$item.Description}}</td>
        <td class="c">{{$item.Unit}}</td>
        <td class="c">{{fmtQty $item.Quantity}}</td>
        <td class="r mono">{{money $item.UnitPrice}}</td>
        <td class="r mono" style="font-weight:600">{{money $item.Amount}}</td>
      </tr>
      {{end}}
    </tbody>
  </table>

  <div class="totals">
    <table class="sum-tbl">
      <tr><td class="slbl">ยอดรวมสินค้า</td><td class="sv">{{money .Doc.Subtotal}}</td></tr>
      <tr style="border-top:.5px solid #ccc">
        <td class="slbl">ส่วนลดรวม</td>
        <td class="sv">{{if gt .Doc.TotalDiscount 0.0}}-{{money .Doc.TotalDiscount}}{{else}}0{{end}}</td>
      </tr>
      {{if gt .Doc.VatRate 0.0}}
      <tr style="border-top:.5px solid #ccc"><td class="slbl">ภาษีมูลค่าเพิ่ม {{fmtQty .Doc.VatRate}}%</td><td class="sv">{{money .Doc.VatAmount}}</td></tr>
      {{end}}
      <tr class="total-row"><td>ยอดที่ต้องชำระ</td><td class="sv">{{money .Doc.TotalAmount}}</td></tr>
    </table>
  </div>

  <div class="notes">
    <div class="notes-lbl">หมายเหตุ</div>
    <div class="notes-box"><span class="notes-content">{{if .Doc.Notes}}{{derefStr .Doc.Notes}}{{end}}</span></div>
  </div>

  <div class="footer">
    <span>ต้นฉบับ — ใบวางบิล เลขที่ {{.Doc.DocumentNo}}</span>
    {{if .Store.TaxID}}<span>TIN: {{.Store.TaxID}}</span>{{end}}
    <span>{{thaiDate .Doc.DocumentDate}}</span>
  </div>
</div>
</body>
</html>`))
