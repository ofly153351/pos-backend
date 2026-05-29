package dochtml

import (
	"fmt"
	"html/template"
)

var whtTmpl = template.Must(template.New("wht").Funcs(template.FuncMap{
	"money":    formatMoney,
	"thaiDate": fmtThaiDate,
	"fmtRate":  func(f float64) string { return fmt.Sprintf("%.0f", f) },
}).Parse(`<!DOCTYPE html>
<html lang="th">
<head>
<meta charset="UTF-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1"/>
<title>{{.FormNo}} — {{.DocumentNo}}</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:'Sarabun','Tahoma',sans-serif;color:#000;background:#fff;font-size:10pt}
@media screen{.page{width:210mm;min-height:240mm;background:#fff;margin:0 auto;padding:12mm 18mm}}
@media print{.page{width:210mm;min-height:240mm;padding:12mm 18mm}@page{size:A4 portrait;margin:0}}
.form-title{text-align:center;font-size:14pt;font-weight:700;margin-bottom:1mm}
.form-sub{text-align:center;font-size:8.5pt;margin-bottom:3mm}
.form-ref{text-align:center;font-size:9pt;font-weight:700;margin-bottom:5mm;border:1.5px solid #000;display:inline-block;padding:1mm 8mm}
.form-ref-wrap{text-align:center;margin-bottom:4mm}
.rule{border:none;border-top:1.5px solid #000;margin-bottom:5mm}
.two-col{display:flex;gap:10mm;margin-bottom:5mm}
.col{flex:1}
.sec-lbl{font-size:7pt;text-transform:uppercase;letter-spacing:.5px;font-weight:700;margin-bottom:1mm;color:#444}
.name{font-size:10.5pt;font-weight:700}
.sub{font-size:8.5pt;margin-top:.5mm;line-height:1.5}
.mono{font-family:monospace;font-weight:600}
.warn{font-style:italic}
.tbl{width:100%;border-collapse:collapse;margin-bottom:5mm;font-size:9pt}
.tbl th{padding:2mm 3mm;border:1px solid #000;font-weight:700;text-align:center}
.tbl td{padding:2mm 3mm;border:1px solid #000;text-align:center}
.tbl td.l{text-align:left}
.tbl td.r{text-align:right;font-family:monospace}
.total-row td{font-weight:700;border-top:1.5px solid #000}
.note-box{border:1px solid #000;padding:3mm 4mm;font-size:8.5pt;margin-bottom:5mm}
.sig-area{display:flex;gap:10mm;margin-top:8mm}
.sig-box{flex:1;text-align:center}
.sig-line{border-top:1px solid #000;padding-top:1.5mm;margin-top:14mm;font-size:8pt}
.sig-sub{font-size:7.5pt;color:#333;margin-top:.5mm}
.footer{margin-top:5mm;border-top:.5px solid #888;padding-top:2mm;font-size:7pt;color:#444;display:flex;justify-content:space-between}
</style>
</head>
<body>
<div class="page">
  <div class="form-title">หนังสือรับรองการหักภาษี ณ ที่จ่าย</div>
  <div class="form-sub">ตามมาตรา 50 ทวิ แห่งประมวลรัษฎากร</div>
  <div class="form-ref-wrap"><span class="form-ref">{{.FormNo}}</span></div>

  <hr class="rule"/>

  <div class="two-col">
    <div class="col">
      <div class="sec-lbl">ผู้จ่ายเงิน / Payer</div>
      <div class="name">{{.PayerName}}</div>
      {{if .PayerAddress}}<div class="sub">{{.PayerAddress}}</div>{{end}}
      <div class="sub">เลขประจำตัวผู้เสียภาษี: {{if .PayerTaxID}}<span class="mono">{{.PayerTaxID}}</span>{{else}}<span class="warn">ยังไม่ได้ตั้งค่า</span>{{end}}</div>
    </div>
    <div class="col">
      <div class="sec-lbl">ผู้รับเงิน / Payee</div>
      <div class="name">{{.PayeeName}}</div>
      {{if .PayeeAddress}}<div class="sub">{{.PayeeAddress}}</div>{{end}}
      {{if .PayeeTaxID}}<div class="sub">เลขประจำตัวผู้เสียภาษี: <span class="mono">{{.PayeeTaxID}}</span></div>{{end}}
    </div>
  </div>

  <table class="tbl">
    <thead>
      <tr>
        <th style="width:10mm">#</th>
        <th>ประเภทเงินได้ที่จ่าย</th>
        <th style="width:26mm">วันที่จ่าย</th>
        <th style="width:30mm">จำนวนเงินที่จ่าย</th>
        <th style="width:16mm">อัตราภาษี</th>
        <th style="width:30mm">ภาษีที่หักและนำส่ง</th>
      </tr>
    </thead>
    <tbody>
      <tr>
        <td>1</td>
        <td class="l">{{.IncomeType}}{{if .IncomeDesc}}<br/><span style="font-size:8pt;color:#333">{{.IncomeDesc}}</span>{{end}}</td>
        <td>{{thaiDate .PaymentDate}}</td>
        <td class="r">{{money .GrossAmount}}</td>
        <td>{{fmtRate .WHTRate}}%</td>
        <td class="r">{{money .WHTAmount}}</td>
      </tr>
    </tbody>
    <tfoot>
      <tr class="total-row">
        <td colspan="3">รวม</td>
        <td class="r">{{money .GrossAmount}}</td>
        <td></td>
        <td class="r">{{money .WHTAmount}}</td>
      </tr>
    </tfoot>
  </table>

  <div class="note-box">
    <strong>เงินสุทธิที่จ่าย:</strong> {{money .NetAmount}} บาท &nbsp;·&nbsp;
    <strong>อ้างอิงเอกสาร:</strong> {{.DocumentNo}}
  </div>

  <div class="note-box" style="font-size:8pt">
    ผู้จ่ายเงินได้หักภาษี ณ ที่จ่ายไว้ถูกต้องแล้ว และขอรับรองว่าจะนำส่งภาษีที่หักนี้ต่อกรมสรรพากรตามกฎหมาย
    ออกให้ ณ วันที่ {{thaiDate .PaymentDate}}
  </div>

  <div class="sig-area">
    <div class="sig-box">
      <div class="sig-line">ลายมือชื่อผู้จ่ายเงิน</div>
      <div class="sig-sub">{{.PayerName}}</div>
    </div>
    <div class="sig-box">
      <div class="sig-line">วันเดือนปีที่ออกหนังสือรับรอง</div>
      <div class="sig-sub">{{thaiDate .PaymentDate}}</div>
    </div>
  </div>

  <div class="footer">
    <span>{{.FormNo}} · อ้างอิง: {{.DocumentNo}}</span>
    {{if .PayerTaxID}}<span>TIN ผู้จ่าย: {{.PayerTaxID}}</span>{{end}}
    <span>{{thaiDate .PaymentDate}}</span>
  </div>
</div>
</body>
</html>`))
