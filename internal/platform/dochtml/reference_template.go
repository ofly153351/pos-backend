package dochtml

import (
	"bytes"
	"html/template"
)

// RenderReferenceDocumentCopies uses the supplied quotation's A4 geometry for
// every document type. Values come only from the normal document view model;
// copies and physical pagination still use the shared Paged.js pipeline.
func RenderReferenceDocumentCopies(d DocData, store StoreInfo, copyIdx int) (string, error) {
	return renderDocumentCopiesWithTemplate(d, store, copyIdx, referenceTmpl)
}

func RenderReferenceDocumentHTML(d DocData, store StoreInfo) (string, error) {
	var buf bytes.Buffer
	vm := BuildDocumentView(d, store)
	vm.ShowSignature = true
	if err := referenceTmpl.Execute(&buf, vm); err != nil {
		return "", err
	}
	return buf.String(), nil
}

var referenceTmpl = template.Must(template.New("reference-document").Parse(referenceDocumentHTML))

// Ref coordinates derive from quotation-recreated.html: 23.5mm left edge,
// 161.5mm content width, 73/27 customer split, blue table head, black grid.
// The sample's fixed 142.7mm item-and-total block is approximated on short
// sheets with a small print safety margin; long item lists paginate naturally.
const referenceDocumentHTML = `{{$root := .}}<!doctype html><html lang="th"><head><meta charset="utf-8"><title>{{.TitleTH}} {{.DocNo}}</title><style>
:root{--ink:#080808;--line:#080808;--row-h:7.5mm}
*{box-sizing:border-box}
@page{size:A4;margin:21.5mm 25mm 26mm 23.5mm}
@page:first{margin-top:21.5mm}
@page signed-document{size:A4;margin:21.5mm 25mm 26mm 23.5mm}
@page signed-document:first{margin-top:21.5mm}
body{margin:0;color:var(--ink);font-family:'Sarabun','Tahoma',sans-serif;font-size:11px;background:#fff;-webkit-print-color-adjust:exact;print-color-adjust:exact}
.page{width:161.5mm;margin:0;position:relative;page-break-inside:auto}
.page[data-item-count="1"] .items,.page[data-item-count="2"] .items,.page[data-item-count="3"] .items,.page[data-item-count="4"] .items{min-height:var(--ref-item-min-height)}
.page[data-compact="true"][data-item-count="1"] .items,.page[data-compact="true"][data-item-count="2"] .items,.page[data-compact="true"][data-item-count="3"] .items,.page[data-compact="true"][data-item-count="4"] .items{min-height:min(100mm,var(--ref-item-min-height))}
.page[data-dense="true"][data-item-count="1"] .items,.page[data-dense="true"][data-item-count="2"] .items,.page[data-dense="true"][data-item-count="3"] .items,.page[data-dense="true"][data-item-count="4"] .items{min-height:60mm}
.page[data-show-signature="true"]{page:signed-document}
.copy-break{break-after:page;page-break-after:always}.copy-break:last-child{break-after:auto;page-break-after:auto}
@media screen{body{background:#e2e8f0}.pagedjs_pages{display:flex;flex-direction:column;align-items:center;gap:16mm;padding:8mm;background:#e2e8f0}.pagedjs_page{background:#fff;box-shadow:0 2px 8px #0002}.copy-break:not(:last-child){margin-bottom:16mm;border-bottom:2px dashed #94a3b8}}
@media print{body{background:#fff}.copy-break{margin:0}}
.screen-page-number,.pos-page-number{position:absolute;right:12mm;top:8mm;font-size:9px}.screen-page-number{right:0;top:-12mm}@media print{.screen-page-number{display:none}}
.pagedjs_page{position:relative}
.hdr-mini{display:none}.pos-continuation-header{position:absolute;top:14mm;left:23.5mm;right:25mm;display:flex;justify-content:space-between;border-bottom:1px solid #080808;padding-bottom:2mm;font-size:10px}
.ref-header{display:flex;align-items:flex-start;justify-content:space-between;min-height:32mm;break-inside:avoid}
.ref-store{display:flex;align-items:flex-start;gap:3mm;padding-left:2.5mm;min-width:0;max-width:112mm}.logo{width:29mm;height:29mm;object-fit:contain;flex:none}
.store-name{font-size:20pt;font-weight:700;line-height:1.05}.store-meta{font-size:8pt;line-height:1.3;overflow-wrap:anywhere}.store-contact{font-size:10pt;font-weight:700;border-bottom:1px solid #111}
.ref-title{text-align:center;min-width:48mm;max-width:50mm;font-size:20pt;font-weight:700;line-height:1.1}.ref-title .english{display:block;font-size:15pt}.ref-title-long{font-size:16pt}.ref-title-long .english{font-size:12pt}.ref-badge{font-size:8pt;font-weight:400;margin-top:2mm}
.ref-customer{border:2px solid #000;margin-top:1.7mm;display:grid;grid-template-columns:73% 27%;min-height:25.8mm;break-inside:avoid}.ref-customer-main{padding:2mm 3mm;font-size:10pt;overflow-wrap:anywhere}.ref-customer-main .line{margin-bottom:1.2mm}.ref-customer-main .tax{margin-top:.6mm}.ref-meta{border-left:1.5px solid #000;padding:1.6mm;font-size:9pt;overflow-wrap:anywhere}.ref-meta div{margin-bottom:1.2mm}
.ref-intro{margin:1.3mm 1mm 2mm;font-size:10pt;break-inside:avoid}.ref-intro .en{margin-top:1mm}.ref-intro strong{font-weight:700}
.items,.bill-items{width:100%;border:2px solid #000;border-collapse:collapse;table-layout:fixed}.items th,.items td,.bill-items th,.bill-items td{border-left:1.5px solid #000;border-right:1.5px solid #000;padding:1.2mm 1mm;vertical-align:top;overflow-wrap:anywhere}.items th,.bill-items th{background:#b6cce3;text-align:center;font-weight:400;font-size:10pt;border-bottom:1.5px solid #000;line-height:1.45}.items td,.bill-items td{font-size:10pt}.items tr,.bill-items tr{break-inside:avoid;page-break-inside:avoid}thead{display:table-header-group}table[data-split-from] thead,table[data-split-from] thead tr,table[data-split-from] thead th{visibility:visible!important;opacity:1!important}
.col-no{width:7.5%}.col-qty{width:15%}.col-price{width:11.5%}.col-amt{width:15.5%}.col-unit{width:12%}.col-disc{width:12%}.col-desc{width:auto}
.items[data-split-from],.bill-items[data-split-from]{width:100%!important;table-layout:fixed!important}.items[data-split-from] .col-no{width:7.5%!important}.items[data-split-from] .col-qty{width:15%!important}.items[data-split-from] .col-price{width:11.5%!important}.items[data-split-from] .col-amt{width:15.5%!important}
.num{text-align:right}.center{text-align:center}.left{text-align:left}.desc{line-height:1.8}.desc .en{font-size:9pt}
.ref-end{break-inside:avoid;page-break-inside:avoid}.ref-total{border:2px solid #000;border-top:0}.ref-total-row{display:grid;grid-template-columns:84.5% 15.5%;min-height:7.5mm;border-top:1.5px solid #000;font-size:10pt;font-weight:600}.ref-total-row span{padding:1.5mm 2mm}.ref-total-row span:first-child{padding-left:13mm}.ref-total-row span:last-child{text-align:right;border-left:1.5px solid #000}.ref-words{border-top:1.5px solid #000;text-align:center;padding:1.5mm;font-size:10pt}
.ref-terms{display:grid;grid-template-columns:68% 32%;border:2px solid #000;margin-top:4mm;min-height:29.2mm;break-inside:avoid}.ref-terms-list{display:grid;grid-template-rows:repeat(3,minmax(8.4mm,auto))}.ref-term{display:grid;grid-template-columns:44% 56%;border-bottom:1.5px solid #000}.ref-term:last-child{border-bottom:0}.ref-term span{padding:2mm 1mm;overflow-wrap:anywhere}.ref-term span:first-child{border-right:1.5px solid #000}.ref-sign{border-left:1.5px solid #000;text-align:center;padding:3mm 2mm 1mm;font-size:9pt;display:flex;flex-direction:column;justify-content:space-between}
.signature-source{display:none!important}.pos-page-signatures{position:absolute;left:23.5mm;right:25mm;bottom:16mm;margin:0;z-index:1}.signatures{display:flex;justify-content:space-around;gap:8mm;break-inside:avoid}.sig{width:48mm;text-align:center}.sig-t{font-size:9pt;border-bottom:1px solid #000}.sig-write{margin-top:4mm}.sig-ln{display:inline-block;width:30mm;border-bottom:1px solid #000}.sig-date-row{margin-top:2mm;font-size:8pt}.sig-date-seg{display:inline-block;width:7mm;border-bottom:1px solid #000}
</style></head><body><div class="page" style="--ref-item-min-height:{{.ReferenceItemMinHeight}}" data-copy-purpose="{{.Purpose}}" data-compact="{{if .ReferenceCompact}}true{{else}}false{{end}}" data-dense="{{if .ReferenceDense}}true{{else}}false{{end}}" data-item-count="{{.ItemCount}}" data-show-signature="{{if .ShowSignature}}true{{else}}false{{end}}">
<div class="screen-page-number">หน้า 1</div><div class="hdr-mini"><span class="m-store">{{.StoreName}}</span><span class="m-title">{{.TitleTH}} · {{.DocNo}} (ต่อ / cont.)</span></div>
<header class="ref-header"><div class="ref-store">{{if .LogoURL}}<img class="logo" src="{{.LogoURL}}" alt="logo">{{end}}<div><div class="store-name">{{.StoreName}}</div>{{if .StoreBranch}}<div class="store-meta">({{.StoreBranch}})</div>{{end}}{{if .StoreAddr}}<div class="store-meta">{{.StoreAddr}}</div>{{end}}{{if .StorePhone}}<div class="store-meta store-contact">TEL: {{.StorePhone}}</div>{{end}}{{if .StoreTaxID}}<div class="store-meta">เลขประจำตัวผู้เสียภาษี {{.StoreTaxID}}</div>{{end}}{{if .StorePreviewContact}}<div class="store-meta">{{.StorePreviewContact}}</div>{{end}}</div></div><div class="ref-title{{if eq .TitleEN "Delivery Note / Tax Invoice"}} ref-title-long{{end}}">{{.TitleTH}}<span class="english">{{.TitleEN}}</span>{{if .Badge}}<div class="ref-badge">{{.Badge}}</div>{{end}}</div></header>
<section class="ref-customer"><div class="ref-customer-main"><div class="line">เรียน&nbsp; {{.CustomerName}}</div><div class="line">ที่อยู่&nbsp; {{if .CustomerAddr}}{{.CustomerAddr}}{{else}}-{{end}}</div>{{if .CustomerPhone}}<div class="line">โทร.&nbsp; {{.CustomerPhone}}</div>{{end}}{{if .DeliveryAddr}}<div class="line">ที่อยู่จัดส่ง&nbsp; {{.DeliveryAddr}}</div>{{end}}{{if .DeliveryContact}}<div class="line">ผู้ติดต่อ&nbsp; {{.DeliveryContact}}</div>{{end}}{{if .DeliveryPhone}}<div class="line">โทร.จัดส่ง&nbsp; {{.DeliveryPhone}}</div>{{end}}{{if .CustomerTaxID}}<div class="tax">เลขประจำตัวผู้เสียภาษี&nbsp; {{.CustomerTaxID}}</div>{{end}}</div><div class="ref-meta"><div><b>No. {{.DocNo}}</b></div><div>Date : {{.DocDate}}</div>{{if .QuotationRefNo}}<div>Ref. {{.QuotationRefNo}}</div>{{end}}{{if .RefNo}}<div>{{.RefLabel}}: {{.RefNo}}</div>{{end}}{{if .PORefNo}}<div>PO: {{.PORefNo}}</div>{{end}}{{if .SpecialValue}}<div>{{.SpecialLabel}}: {{.SpecialValue}}</div>{{end}}{{if .StaffName}}<div>ผู้จัดทำ: {{.StaffName}}</div>{{end}}</div></section>
<div class="ref-intro">{{if eq .TitleEN "Quotation"}}<div>เรามีความยินดี ขอเสนอราคาผลิตภัณฑ์ โดยมีรายละเอียดดังนี้</div><div class="en">We are pleased to quote you the price of products as follows</div>{{else}}<div>{{.TitleTH}} / {{.TitleEN}}</div>{{end}}</div>
{{if eq .TitleTH "ใบวางบิล"}}<table class="bill-items"><thead><tr><th class="col-no">ลำดับ<br>Item</th><th>ใบส่งสินค้า<br>Delivery Order</th><th>วันที่ออก<br>Date</th><th>ครบกำหนด<br>Due</th><th>จำนวนเงิน<br>Amount</th></tr></thead><tbody>{{range .BillRows}}<tr><td class="center">{{.No}}</td><td>{{.DocumentNo}}</td><td>{{.IssueDate}}</td><td>{{.DueDate}}</td><td class="num">{{.Amount}}</td></tr>{{end}}</tbody></table>{{else}}<table class="items"><colgroup><col class="col-no"><col class="col-desc"><col class="col-qty"><col class="col-price">{{if .ShowReferenceDiscount}}<col class="col-disc">{{end}}<col class="col-amt"></colgroup><thead><tr><th class="col-no">ลำดับ<br>Item</th><th class="col-desc">รายการ<br>Description</th><th class="col-qty">จำนวน<br>Quantity</th><th class="col-price">หน่วยละ<br>Unit Price</th>{{if .ShowReferenceDiscount}}<th class="col-disc">ส่วนลด<br>Discount</th>{{end}}<th class="col-amt">จำนวนเงิน<br>Amount</th></tr></thead><tbody>{{range $pg := .Pages}}{{range $it := $pg.Items}}<tr><td class="center col-no">{{$it.No}}</td><td class="col-desc"><div class="desc">{{$it.Desc}}{{if $it.DescEn}}<br><span class="en">{{$it.DescEn}}</span>{{end}}</div></td><td class="num col-qty">{{$it.Qty}} {{$it.Unit}}</td><td class="num col-price">{{$it.UnitPrice}}</td>{{if $root.ShowReferenceDiscount}}<td class="num col-disc">{{if $it.HasDiscount}}{{$it.Discount}}{{else}}-{{end}}</td>{{end}}<td class="num col-amt">{{$it.Amount}}</td></tr>{{end}}{{end}}</tbody></table>{{end}}
<div class="ref-end"><div class="ref-total">{{range .Summary}}<div class="ref-total-row"><span>{{.Label}}</span><span>{{.Value}}</span></div>{{end}}<div class="ref-words">จำนวนเงินตัวอักษร&nbsp; ( {{.ReferenceBahtText}} )</div></div>
<div class="ref-terms"><div class="ref-terms-list">{{if .ShowReferenceTerms}}<div class="ref-term"><span>กำหนดยืนราคา Price Validity</span><span>{{.PriceValidity}}</span></div><div class="ref-term"><span>กำหนดส่งมอบ Delivery Time</span><span>{{.DeliveryLeadTime}}</span></div><div class="ref-term"><span>เงื่อนไขการชำระเงิน Payment</span><span>{{.PaymentTerms}}</span></div>{{else}}<div class="ref-term"><span>เลขที่เอกสาร Document No.</span><span>{{.DocNo}}</span></div><div class="ref-term"><span>วันที่ Date</span><span>{{.DocDate}}</span></div>{{if .SpecialValue}}<div class="ref-term"><span>{{.SpecialLabel}}</span><span>{{.SpecialValue}}</span></div>{{end}}{{end}}{{if .Notes}}<div class="ref-term"><span>หมายเหตุ Remarks</span><span>{{.Notes}}</span></div>{{end}}{{range .Banks}}<div class="ref-term"><span>โอนเงิน Bank Transfer</span><span>{{.Name}} {{.No}} {{.Holder}}</span></div>{{end}}</div><div class="ref-sign">{{if .ShowSignature}}<span>{{.SigRightTH}}</span><span>{{.StaffName}}</span><span>{{.SigRightEN}}</span>{{end}}</div></div></div>
</div></body></html>`
