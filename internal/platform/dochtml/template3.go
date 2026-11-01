package dochtml

import (
	"bytes"
	"html/template"
)

// RenderTemplate3DocumentCopies renders the monochrome quotation-style A4 layout
// using the same persisted document data, copy variants and Paged.js pipeline.
func RenderTemplate3DocumentCopies(d DocData, store StoreInfo, copyIdx int) (string, error) {
	return renderDocumentCopiesWithTemplate(d, store, copyIdx, template3Tmpl)
}

func RenderTemplate3DocumentHTML(d DocData, store StoreInfo) (string, error) {
	var buf bytes.Buffer
	if err := template3Tmpl.Execute(&buf, BuildDocumentView(d, store)); err != nil {
		return "", err
	}
	return buf.String(), nil
}

var template3Tmpl = template.Must(template.New("template-3").Parse(template3HTML))

// The supplied index.html embeds only a scanned image. Recreate its geometry
// with semantic HTML; never embed the sample business data or its signatures.
const template3HTML = `{{$root := .}}<!doctype html><html lang="th"><head><meta charset="utf-8"><title>{{.TitleTH}} {{.DocNo}}</title><style>
*{box-sizing:border-box}
@page{size:A4;margin:16mm 17mm 18mm}
@page signed-document{size:A4;margin:16mm 17mm 36mm}
.page[data-show-signature="true"]{page:signed-document}
body{margin:0;background:#fff;color:#111;font:12px/1.45 'Sarabun','Tahoma',sans-serif;-webkit-print-color-adjust:exact;print-color-adjust:exact}
.page{position:relative;width:176mm;margin:0;min-height:0}
.copy-break{break-after:page;page-break-after:always}.copy-break:last-child{break-after:auto;page-break-after:auto}
@media print{.copy-break{margin:0}}
@media screen{body{background:#e2e8f0}.pagedjs_pages{display:flex;flex-direction:column;align-items:center;gap:16mm;padding:8mm}.pagedjs_page{background:#fff;box-shadow:0 2px 8px #0002}.copy-break:not(:last-child){margin-bottom:16mm;border-bottom:2px dashed #94a3b8}}
.pagedjs_page{position:relative}.screen-page-number,.pos-page-number{position:absolute;top:8mm;right:12mm;font-size:9px}.screen-page-number{top:-8mm;right:0}@media print{.screen-page-number{display:none}}
.hdr-mini{display:none}.pos-continuation-header{position:absolute;top:14mm;left:17mm;right:17mm;display:flex;justify-content:space-between;border-bottom:1px solid #111;padding-bottom:2mm;font-size:10px}
.t3-header{display:flex;justify-content:space-between;align-items:stretch;gap:8mm;min-height:68mm;break-inside:avoid}
.t3-store{align-self:flex-start;flex:1;min-width:0}.t3-logo{width:22mm;height:21mm;object-fit:contain;object-position:left center}.t3-name{font-size:17px;line-height:1.25;font-weight:700;overflow-wrap:anywhere}.t3-meta{font-size:11px;line-height:1.5;overflow-wrap:anywhere}
.t3-title{align-self:flex-end;width:75mm;flex:none;border:1px solid #111;text-align:center;padding:4mm 2mm;line-height:1.2}.t3-title strong{display:block;font-size:19px}.t3-title span{display:block;font-size:12px}.t3-badge{font-size:10px;margin-top:1mm}
.t3-id{text-align:right;margin:3mm 0 1mm;font-weight:600}.t3-id span{display:block}.t3-intro{margin:4mm 0 1mm;break-inside:avoid}.t3-intro div{margin:1mm 0}.t3-intro strong{font-weight:700}
.t3-items,.t3-bill{width:100%;table-layout:fixed;border-collapse:collapse;border:1px solid #111}.t3-items th,.t3-items td,.t3-bill th,.t3-bill td{border:1px solid #111;padding:1.5mm 1mm;vertical-align:top;overflow-wrap:anywhere}.t3-items th,.t3-bill th{font-weight:600;text-align:center}.t3-items td,.t3-bill td{font-size:12px}.t3-items tr,.t3-bill tr{break-inside:avoid;page-break-inside:avoid}thead{display:table-header-group}table[data-split-from] thead,table[data-split-from] thead tr,table[data-split-from] thead th{visibility:visible!important;opacity:1!important}
.t3-items .col-no{width:7%}.t3-items .col-desc{width:auto}.t3-items .col-qty{width:10%}.t3-items .col-unit{width:8%}.t3-items .col-price{width:12%}.t3-items .col-discount{width:11%}.t3-items .col-amt{width:15%}.t3-bill .col-no{width:8%}
.t3-items[data-split-from],.t3-bill[data-split-from]{width:100%!important;table-layout:fixed!important}
.page[data-item-count="1"] .t3-items,.page[data-item-count="1"] .t3-bill{min-height:65mm}.page[data-item-count="2"] .t3-items,.page[data-item-count="2"] .t3-bill{min-height:50mm}
.page[data-compact="true"] .t3-header{min-height:58mm}.page[data-compact="true"][data-item-count="1"] .t3-items,.page[data-compact="true"][data-item-count="1"] .t3-bill{min-height:50mm}
.page[data-dense="true"] .t3-header{min-height:45mm}.page[data-dense="true"][data-item-count="1"] .t3-items,.page[data-dense="true"][data-item-count="1"] .t3-bill{min-height:32mm}
.t3-number{text-align:right;white-space:nowrap}.t3-center{text-align:center}.t3-end{break-inside:avoid;page-break-inside:avoid}.t3-summary{display:grid;grid-template-columns:56% 44%;align-items:start;margin-top:2mm;gap:4mm}.t3-words{border:1px solid #111;text-align:center;padding:2mm;min-height:8mm}.t3-words-label{text-align:center;font-size:10px;margin-top:1mm}.t3-total{width:100%;border-collapse:collapse}.t3-total td{padding:1mm 2mm}.t3-total td:last-child{border:1px solid #111;text-align:right;min-width:27mm;font-weight:600}.t3-total tr:last-child{font-weight:700}
.t3-lower{display:grid;grid-template-columns:56% 44%;gap:4mm;margin-top:3mm}.t3-approval-box{border:1px solid #111;min-height:28mm;text-align:center;padding:2mm}.t3-approval-box strong{display:block}.t3-approval-line{border-bottom:1px dotted #111;height:7mm}.t3-terms div{margin-bottom:1.5mm;overflow-wrap:anywhere}.t3-notes{margin:3mm 0;overflow-wrap:anywhere}
.signature-source{display:none!important}.pos-page-signatures{position:absolute;left:17mm;right:17mm;bottom:11mm;z-index:1;margin:0}.t3-signatures{display:flex;justify-content:space-around;gap:12mm;margin-top:4mm}.t3-sig{text-align:center;width:67mm}.t3-sig-line{border-bottom:1px dotted #111;height:8mm}.t3-sig-role{font-size:11px;margin-top:1mm}
</style></head><body><div class="page" data-copy-purpose="{{.Purpose}}" data-item-count="{{.ItemCount}}" data-compact="{{if .ReferenceCompact}}true{{else}}false{{end}}" data-dense="{{if .ReferenceDense}}true{{else}}false{{end}}" data-show-signature="{{if .ShowSignature}}true{{else}}false{{end}}">
<div class="screen-page-number">หน้า 1</div><div class="hdr-mini"><span>{{.StoreName}}</span><span>{{.TitleTH}} · {{.DocNo}} (ต่อ / cont.)</span></div>
<header class="t3-header"><div class="t3-store">{{if .LogoURL}}<img class="t3-logo" src="{{.LogoURL}}" alt="logo">{{end}}<div class="t3-name">{{.StoreName}}</div>{{if .StoreBranch}}<div class="t3-meta">{{.StoreBranch}}</div>{{end}}{{if .StoreAddr}}<div class="t3-meta">{{.StoreAddr}}</div>{{end}}{{if .StorePhone}}<div class="t3-meta">TEL. {{.StorePhone}}</div>{{end}}{{if .StoreTaxID}}<div class="t3-meta">เลขประจำตัวผู้เสียภาษี {{.StoreTaxID}}</div>{{end}}</div><div class="t3-title"><strong>{{.TitleTH}}</strong><span>{{.TitleEN}}</span>{{if .Badge}}<div class="t3-badge">{{.Badge}}</div>{{end}}</div></header>
<div class="t3-id"><span>NO. {{.DocNo}}</span><span>วันที่ {{.DocDate}}</span></div>
<div class="t3-intro">{{if .QuotationSummary}}<div>เรื่อง&nbsp; {{.QuotationSummary}}</div>{{end}}<div>เรียน&nbsp; <strong>{{.CustomerName}}</strong></div>{{if .CustomerAddr}}<div>ที่อยู่&nbsp; {{.CustomerAddr}}</div>{{end}}{{if .CustomerPhone}}<div>โทร.&nbsp; {{.CustomerPhone}}</div>{{end}}{{if .CustomerTaxID}}<div>เลขประจำตัวผู้เสียภาษี&nbsp; {{.CustomerTaxID}}</div>{{end}}{{if .DeliveryAddr}}<div>ที่อยู่จัดส่ง&nbsp; {{.DeliveryAddr}}</div>{{end}}{{if .PORefNo}}<div>PO&nbsp; {{.PORefNo}}</div>{{end}}{{if .RefNo}}<div>{{.RefLabel}}&nbsp; {{.RefNo}}</div>{{end}}{{if .QuotationRefNo}}<div>อ้างอิงใบเสนอราคา&nbsp; {{.QuotationRefNo}}</div>{{end}}{{if .SpecialValue}}<div>{{.SpecialLabel}}&nbsp; {{.SpecialValue}}</div>{{end}}{{if eq .TitleEN "Quotation"}}<div>ทางร้านขอเสนอราคาตามรายการดังต่อไปนี้</div>{{end}}</div>
{{if eq .TitleTH "ใบวางบิล"}}<table class="t3-bill"><thead><tr><th class="col-no">ลำดับ</th><th>ใบส่งสินค้า</th><th>วันที่ออก</th><th>ครบกำหนด</th><th>จำนวนเงิน</th></tr></thead><tbody>{{range .BillRows}}<tr><td class="t3-center">{{.No}}</td><td>{{.DocumentNo}}</td><td>{{.IssueDate}}</td><td>{{.DueDate}}</td><td class="t3-number">{{.Amount}}</td></tr>{{end}}</tbody></table>{{else}}<table class="t3-items"><colgroup><col class="col-no"><col class="col-desc"><col class="col-qty"><col class="col-unit"><col class="col-price">{{if .ShowReferenceDiscount}}<col class="col-discount">{{end}}<col class="col-amt"></colgroup><thead><tr><th>ลำดับ</th><th>รายการ</th><th>จำนวน</th><th>หน่วย</th><th>ราคา/หน่วย</th>{{if .ShowReferenceDiscount}}<th>ส่วนลด</th>{{end}}<th>จำนวนเงิน</th></tr></thead><tbody>{{range $pg := .Pages}}{{range $it := $pg.Items}}<tr><td class="t3-center">{{$it.No}}</td><td>{{$it.Desc}}{{if $it.DescEn}}<br>{{$it.DescEn}}{{end}}</td><td class="t3-number">{{$it.Qty}}</td><td class="t3-center">{{$it.Unit}}</td><td class="t3-number">{{$it.UnitPrice}}</td>{{if $root.ShowReferenceDiscount}}<td class="t3-number">{{if $it.HasDiscount}}{{$it.Discount}}{{else}}-{{end}}</td>{{end}}<td class="t3-number">{{$it.Amount}}</td></tr>{{end}}{{end}}</tbody></table>{{end}}
<div class="t3-end"><div class="t3-summary"><div><div class="t3-words">{{.ReferenceBahtText}}</div><div class="t3-words-label">จำนวนเงินรวมทั้งสิ้น (ตัวอักษร)</div></div><table class="t3-total"><tbody>{{range .Summary}}<tr><td>{{.Label}}</td><td>{{.Value}}</td></tr>{{end}}</tbody></table></div>
<div class="t3-lower">{{if .ShowReferenceTerms}}<div class="t3-approval-box"><strong>สำหรับลูกค้า</strong><div>อนุมัติสั่งซื้อตามใบเสนอราคานี้</div><div class="t3-approval-line"></div><div class="t3-approval-line"></div><div>(ลงชื่อ) ผู้มีอำนาจสั่งซื้อ</div></div>{{else}}<div></div>{{end}}<div class="t3-terms">{{if .ShowReferenceTerms}}<div>กำหนดยืนราคา {{.PriceValidity}}</div><div>กำหนดส่งสินค้า {{.DeliveryLeadTime}}</div><div>เงื่อนไขการชำระเงิน {{.PaymentTerms}}</div>{{else}}{{if .SpecialValue}}<div>{{.SpecialLabel}} {{.SpecialValue}}</div>{{end}}{{end}}{{range .Banks}}<div>โอนเงิน {{.Name}} {{.No}} {{.Holder}}</div>{{end}}</div></div>{{if .Notes}}<div class="t3-notes">หมายเหตุ {{.Notes}}</div>{{end}}
{{if .ShowSignature}}<div class="t3-signatures signature-source" data-copy-purpose="{{.Purpose}}"><div class="t3-sig"><div class="t3-sig-line"></div><div class="t3-sig-role">{{.SigLeftTH}} / {{.SigLeftEN}}</div></div><div class="t3-sig"><div class="t3-sig-line"></div><div class="t3-sig-role">{{.SigRightTH}} / {{.SigRightEN}}</div></div></div>{{end}}</div>
</div></body></html>`
