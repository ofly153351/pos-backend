package document

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"time"
)

type StoreInfo struct {
	Name    string
	Address string
	Phone   string
	TaxID   string
	LogoURL string
}

var docTitleTH = map[DocumentType]string{
	TypeInvoice:    "ใบแจ้งหนี้",
	TypeReceipt:    "ใบเสร็จรับเงิน",
	TypeTaxInvoice: "ใบกำกับภาษี",
	TypeQuotation:  "ใบเสนอราคา",
	TypeBill:       "ใบวางบิล",
	TypeCreditNote: "ใบลดหนี้",
}

var docTitleEN = map[DocumentType]string{
	TypeInvoice:    "Invoice",
	TypeReceipt:    "Receipt",
	TypeTaxInvoice: "Tax Invoice",
	TypeQuotation:  "Quotation",
	TypeBill:       "Bill",
	TypeCreditNote: "Credit Note",
}

func formatMoney(f float64) string {
	s := fmt.Sprintf("%.2f", f)
	// Insert comma grouping before the decimal
	parts := strings.SplitN(s, ".", 2)
	intPart := parts[0]
	sign := ""
	if strings.HasPrefix(intPart, "-") {
		sign = "-"
		intPart = intPart[1:]
	}
	var result []byte
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return sign + string(result) + "." + parts[1]
}

func fmtThaiDate(t time.Time) string {
	months := [...]string{
		"", "มกราคม", "กุมภาพันธ์", "มีนาคม", "เมษายน", "พฤษภาคม", "มิถุนายน",
		"กรกฎาคม", "สิงหาคม", "กันยายน", "ตุลาคม", "พฤศจิกายน", "ธันวาคม",
	}
	return fmt.Sprintf("%d %s %d", t.Day(), months[t.Month()], t.Year()+543)
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return fmtThaiDate(*t)
}

func isTaxDoc(t DocumentType) bool {
	return t == TypeTaxInvoice || t == TypeInvoice
}

var printTmpl = template.Must(template.New("doc").Funcs(template.FuncMap{
	"money":     formatMoney,
	"thaiDate":  fmtThaiDate,
	"derefStr":  derefStr,
	"derefTime": derefTime,
	"titleTH":   func(t DocumentType) string { return docTitleTH[t] },
	"titleEN":   func(t DocumentType) string { return docTitleEN[t] },
	"isTaxDoc":  isTaxDoc,
	"inc":       func(i int) int { return i + 1 },
	"fmtQty":    func(f float64) string { return fmt.Sprintf("%.0f", f) },
}).Parse(`<!DOCTYPE html>
<html lang="th">
<head>
<meta charset="UTF-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1"/>
<title>{{titleTH .Doc.Type}} — {{.Doc.DocumentNo}}</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:'Sarabun','Tahoma',sans-serif;color:#000;background:#fff;font-size:10pt}
@media screen{.page{width:210mm;min-height:297mm;background:#fff;margin:0 auto;padding:15mm 18mm}}
@media print{.page{width:210mm;min-height:297mm;padding:15mm 18mm}@page{size:A4 portrait;margin:0}}
.hdr{display:flex;justify-content:space-between;align-items:flex-start;margin-bottom:5mm}
.store-name{font-size:14pt;font-weight:700;margin-bottom:1.5mm}
.meta-line{font-size:8.5pt;line-height:1.6}
.warn{font-style:italic}
.doc-side{text-align:right}
.doc-sub{font-size:7.5pt;font-weight:600;letter-spacing:.5px;margin-bottom:1mm;border:1px solid #000;display:inline-block;padding:0.5mm 3mm;margin-bottom:2mm}
.doc-title{font-size:20pt;font-weight:800;margin-bottom:2.5mm}
.doc-info{font-size:8.5pt;border-collapse:collapse;margin-left:auto}
.doc-info td{padding:.8mm 2mm}
.doc-info .lbl{color:#444}
.doc-info .val{font-weight:600;font-family:monospace}
.rule{border:none;border-top:1.5px solid #000;margin-bottom:5mm}
.rule-thin{border:none;border-top:.5px solid #888;margin-bottom:4mm}
.two-col{display:flex;gap:8mm;margin-bottom:6mm}
.col{flex:1}
.sec-lbl{font-size:7pt;text-transform:uppercase;letter-spacing:.5px;margin-bottom:1mm;color:#444;font-weight:600}
.cust-name{font-size:10.5pt;font-weight:700}
.cust-sub{font-size:8.5pt;margin-top:.5mm;line-height:1.5}
.mono{font-family:monospace}
.items{width:100%;border-collapse:collapse;margin-bottom:5mm;font-size:8.5pt}
.items th{padding:1.5mm 2.5mm;font-weight:700;border-top:1.5px solid #000;border-bottom:1px solid #000;text-align:left}
.items td{padding:1.5mm 2.5mm;border-bottom:.5px solid #ccc}
.r{text-align:right}.c{text-align:center}
.items tbody tr:last-child td{border-bottom:1px solid #000}
.tarea{display:flex;gap:8mm;margin-bottom:8mm}
.notes{flex:1;font-size:8.5pt;color:#333}
.notes-lbl{font-weight:600;margin-bottom:1mm;font-size:7.5pt}
.sumtbl{min-width:72mm;font-size:8.5pt;border-collapse:collapse;margin-left:auto}
.sumtbl td{padding:1mm 2.5mm}
.sumtbl .slbl{color:#333}
.sumtbl .sv{text-align:right;font-family:monospace}
.total-row td{border-top:1.5px solid #000;font-size:11pt;font-weight:700;padding:1.5mm 2.5mm}
.sig-area{display:flex;gap:10mm;margin-top:10mm}
.sig-box{flex:1;text-align:center}
.sig-line{border-top:1px solid #000;padding-top:1.5mm;margin-top:14mm;font-size:8pt}
.sig-name{font-size:7.5pt;color:#333;margin-top:.5mm}
.footer{margin-top:5mm;border-top:.5px solid #888;padding-top:2mm;display:flex;justify-content:space-between;font-size:7pt;color:#444}
</style>
</head>
<body>
<div class="page">
  <div class="hdr">
    <div>
      {{if .Store.LogoURL}}<div style="margin-bottom:3mm"><img src="{{.Store.LogoURL}}" alt="logo" style="max-height:40px;max-width:120px;object-fit:contain;display:block"></div>{{end}}
      <div class="store-name">{{.Store.Name}}</div>
      {{if .Store.Address}}<div class="meta-line">{{.Store.Address}}</div>{{end}}
      {{if .Store.Phone}}<div class="meta-line">โทร: {{.Store.Phone}}</div>{{end}}
      <div class="meta-line">เลขประจำตัวผู้เสียภาษี: {{if .Store.TaxID}}<span class="mono">{{.Store.TaxID}}</span>{{else}}<span class="warn">ยังไม่ได้ตั้งค่า</span>{{end}}</div>
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
      {{range $i, $item := .Doc.Items}}
      <tr>
        <td class="c">{{inc $i}}</td>
        <td>{{$item.Description}}</td>
        <td class="c">{{$item.Unit}}</td>
        <td class="c">{{fmtQty $item.Quantity}}</td>
        <td class="r mono">{{money $item.UnitPrice}}</td>
        <td class="r mono">{{if gt $item.DiscountValue 0.0}}-{{money $item.DiscountValue}}{{else}}—{{end}}</td>
        <td class="r mono" style="font-weight:600">{{money $item.Amount}}</td>
      </tr>
      {{end}}
      {{range .FillerRows}}<tr><td>&nbsp;</td><td></td><td></td><td></td><td></td><td></td><td></td></tr>{{end}}
    </tbody>
  </table>

  <div class="tarea">
    <div class="notes">
      {{if .Doc.Notes}}<div class="notes-lbl">หมายเหตุ</div><div>{{derefStr .Doc.Notes}}</div>{{end}}
    </div>
    <table class="sumtbl">
      <tr><td class="slbl">ยอดรวมสินค้า</td><td class="sv">{{money .Doc.Subtotal}}</td></tr>
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
</div>
</body>
</html>`))

type printData struct {
	Doc        *Document
	Store      StoreInfo
	FillerRows []struct{}
}

func RenderDocumentHTML(doc *Document, store StoreInfo) (string, error) {
	if doc.Type == TypeBill {
		return renderBillHTML(doc, store)
	}
	fillerCount := 5 - len(doc.Items)
	if fillerCount < 0 {
		fillerCount = 0
	}
	data := printData{
		Doc:        doc,
		Store:      store,
		FillerRows: make([]struct{}, fillerCount),
	}
	var buf bytes.Buffer
	if err := printTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func renderBillHTML(doc *Document, store StoreInfo) (string, error) {
	var buf bytes.Buffer
	if err := billTmpl.Execute(&buf, printData{Doc: doc, Store: store}); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// billTmpl — ใบวางบิล: formal monochrome layout, payment box = bordered box
var billTmpl = template.Must(template.New("bill").Funcs(template.FuncMap{
	"money":     formatMoney,
	"thaiDate":  fmtThaiDate,
	"derefStr":  derefStr,
	"derefTime": derefTime,
	"inc":       func(i int) int { return i + 1 },
	"fmtQty":    func(f float64) string { return fmt.Sprintf("%.0f", f) },
}).Parse(`<!DOCTYPE html>
<html lang="th">
<head>
<meta charset="UTF-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1"/>
<title>ใบวางบิล — {{.Doc.DocumentNo}}</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:'Sarabun','Tahoma',sans-serif;color:#000;background:#fff;font-size:10pt}
@media screen{.page{width:210mm;min-height:297mm;background:#fff;margin:0 auto;padding:13mm 18mm}}
@media print{.page{width:210mm;min-height:297mm;padding:13mm 18mm}@page{size:A4 portrait;margin:0}}
.hdr{display:flex;justify-content:space-between;align-items:flex-start;margin-bottom:5mm}
.store-name{font-size:14pt;font-weight:700}
.meta-line{font-size:8.5pt;line-height:1.6;margin-top:.5mm}
.warn{font-style:italic}
.doc-side{text-align:right}
.doc-title{font-size:18pt;font-weight:800;margin-bottom:2mm}
.doc-sub{font-size:7.5pt;font-weight:600;border:1px solid #000;display:inline-block;padding:.5mm 3mm;margin-bottom:2mm;letter-spacing:.5px}
.doc-meta{font-size:8.5pt;margin-top:1mm}
.mono{font-family:monospace;font-weight:600}
.rule{border:none;border-top:1.5px solid #000;margin:4mm 0}
/* ── Payment notice box ── */
.pay-box{border:1.5px solid #000;padding:4mm 6mm;margin-bottom:6mm;display:flex;justify-content:space-between;align-items:center}
.pay-left .lbl{font-size:7.5pt;color:#444;margin-bottom:1mm}
.pay-left .due{font-size:12pt;font-weight:700}
.pay-right{text-align:right}
.pay-right .lbl{font-size:7.5pt;color:#444;margin-bottom:1mm}
.pay-right .amount{font-size:20pt;font-weight:800;font-family:monospace}
/* ── Two columns ── */
.two-col{display:flex;gap:8mm;margin-bottom:5mm}
.col{flex:1}
.sec-lbl{font-size:7pt;text-transform:uppercase;letter-spacing:.5px;font-weight:700;margin-bottom:1mm;color:#444}
.cust-name{font-size:10.5pt;font-weight:700}
.cust-sub{font-size:8.5pt;margin-top:.5mm;line-height:1.5}
/* ── Items ── */
.items{width:100%;border-collapse:collapse;margin-bottom:5mm;font-size:8.5pt}
.items th{padding:1.5mm 2.5mm;font-weight:700;border-top:1.5px solid #000;border-bottom:1px solid #000}
.items td{padding:1.5mm 2.5mm;border-bottom:.5px solid #ccc}
.items tbody tr:last-child td{border-bottom:1px solid #000}
.r{text-align:right}.c{text-align:center}
/* ── Totals ── */
.totals{display:flex;justify-content:flex-end;margin-bottom:6mm}
.sum-tbl{min-width:78mm;font-size:8.5pt;border-collapse:collapse}
.sum-tbl td{padding:1mm 2.5mm}
.slbl{color:#333}
.sv{text-align:right;font-family:monospace}
.total-row td{border-top:1.5px solid #000;font-size:11pt;font-weight:700;padding:1.5mm 2.5mm}
.notes{font-size:8.5pt;color:#333;margin-bottom:5mm}
.notes-lbl{font-weight:700;font-size:7.5pt;margin-bottom:1mm}
.footer{border-top:.5px solid #888;padding-top:2mm;display:flex;justify-content:space-between;font-size:7pt;color:#444;margin-top:4mm}
</style>
</head>
<body>
<div class="page">

  <div class="hdr">
    <div>
      {{if .Store.LogoURL}}<div style="margin-bottom:3mm"><img src="{{.Store.LogoURL}}" alt="logo" style="max-height:40px;max-width:120px;object-fit:contain;display:block"></div>{{end}}
      <div class="store-name">{{.Store.Name}}</div>
      {{if .Store.Address}}<div class="meta-line">{{.Store.Address}}</div>{{end}}
      {{if .Store.Phone}}<div class="meta-line">โทร: {{.Store.Phone}}</div>{{end}}
      <div class="meta-line">เลขประจำตัวผู้เสียภาษี: {{if .Store.TaxID}}<span class="mono">{{.Store.TaxID}}</span>{{else}}<span class="warn">ยังไม่ได้ตั้งค่า</span>{{end}}</div>
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
      {{if gt .Doc.VatRate 0.0}}
      <tr style="border-top:.5px solid #ccc"><td class="slbl">ภาษีมูลค่าเพิ่ม {{fmtQty .Doc.VatRate}}%</td><td class="sv">{{money .Doc.VatAmount}}</td></tr>
      {{end}}
      <tr class="total-row"><td>ยอดที่ต้องชำระ</td><td class="sv">{{money .Doc.TotalAmount}}</td></tr>
    </table>
  </div>

  {{if .Doc.Notes}}
  <div class="notes">
    <div class="notes-lbl">หมายเหตุ</div>
    <div>{{derefStr .Doc.Notes}}</div>
  </div>
  {{end}}

  <div class="footer">
    <span>ต้นฉบับ — ใบวางบิล เลขที่ {{.Doc.DocumentNo}}</span>
    {{if .Store.TaxID}}<span>TIN: {{.Store.TaxID}}</span>{{end}}
    <span>{{thaiDate .Doc.DocumentDate}}</span>
  </div>
</div>
</body>
</html>`))

