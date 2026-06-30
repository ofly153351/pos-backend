package warehouse_receipt

import (
	"fmt"
	"html"
	"strings"
	"time"
)

func renderReceiptHTML(receipt WarehouseReceipt) string {
	var rows strings.Builder
	for idx, item := range receipt.Items {
		rows.WriteString(fmt.Sprintf(`
		<tr>
			<td>%d</td>
			<td>%s<br><small>%s</small></td>
			<td>%s</td>
			<td style="text-align:right">%d</td>
			<td style="text-align:right">%.2f</td>
			<td style="text-align:right">%.2f</td>
			<td style="text-align:right">%.2f</td>
		</tr>`, idx+1, html.EscapeString(item.ProductName), html.EscapeString(strings.TrimSpace(item.SKU)), html.EscapeString(strings.TrimSpace(item.LocationName)), item.Quantity, item.UnitPrice, float64(item.Quantity)*item.DiscountAmount, item.LineNet))
	}
	if rows.Len() == 0 {
		rows.WriteString(`<tr><td colspan="7" style="text-align:center">No items</td></tr>`)
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>%s</title>
<style>
body { font-family: Arial, sans-serif; margin: 24px; color: #222; }
.header { display:flex; justify-content:space-between; margin-bottom: 20px; }
.box { border:1px solid #ddd; padding:12px; border-radius:8px; }
table { width:100%%; border-collapse: collapse; margin-top: 16px; }
th, td { border:1px solid #ddd; padding:8px; font-size:12px; vertical-align:top; }
th { background:#f5f5f5; }
.summary { margin-top: 16px; width: 320px; margin-left: auto; }
.summary td { border:none; padding:4px 0; }
.signatures { display:flex; justify-content:center; gap:20mm; margin-top:12mm; }
.sig { width:56mm; text-align:center; }
.sig-t { font-size:9pt; font-weight:700; margin-bottom:3mm; border-bottom:1px solid #000; padding-bottom:1mm; }
/* justify-content:center keeps the signing content centered inside each 56mm column */
.sig-write { display:flex; justify-content:center; align-items:baseline; gap:2mm; margin-bottom:2mm; font-size:9pt; }
.sig-write-lbl { white-space:nowrap; flex-shrink:0; }
.sig-ln { width:40mm; flex-shrink:0; border-bottom:1px solid #000; padding-top:14mm; }
.sig-date-row { display:flex; justify-content:center; align-items:baseline; gap:1mm; font-size:8pt; color:#666; margin-top:1mm; }
.sig-date-seg { width:12mm; border-bottom:1px solid #000; }
.sig-sub { font-size:8pt; color:#555; margin-top:3mm; }
</style>
</head>
<body>
	<div class="header">
		<div>
			<h2>Goods Receipt Note</h2>
			<div><strong>Document No:</strong> %s</div>
			<div><strong>Status:</strong> %s</div>
			<div><strong>Received At:</strong> %s</div>
		</div>
		<div class="box">
			<div><strong>Warehouse:</strong> %s</div>
			<div><strong>Supplier:</strong> %s</div>
			<div><strong>PO:</strong> %s</div>
			<div><strong>Reference:</strong> %s</div>
		</div>
	</div>
	<div class="box">
		<div><strong>Note:</strong> %s</div>
	</div>
	<table>
		<thead>
			<tr>
				<th style="width:48px">#</th>
				<th>Product</th>
				<th>Location</th>
				<th style="text-align:right">Qty</th>
				<th style="text-align:right">Unit Price</th>
				<th style="text-align:right">Discount</th>
				<th style="text-align:right">Line Net</th>
			</tr>
		</thead>
		<tbody>%s
		</tbody>
	</table>
	<table class="summary">
		<tr><td>Subtotal</td><td style="text-align:right">%.2f</td></tr>
		<tr><td>Discount</td><td style="text-align:right">%.2f</td></tr>
		<tr><td>Net</td><td style="text-align:right">%.2f</td></tr>
		<tr><td>VAT %.2f%%</td><td style="text-align:right">%.2f</td></tr>
		<tr><td><strong>Total</strong></td><td style="text-align:right"><strong>%.2f</strong></td></tr>
	</table>
	<div class="signatures">
		<div class="sig">
			<div class="sig-t">ผู้ส่งมอบ / Delivered By</div>
			<div class="sig-write"><span class="sig-write-lbl">ลงชื่อ</span><span class="sig-ln"></span></div>
			<div class="sig-date-row"><span>วันที่</span><span class="sig-date-seg"></span><span>/</span><span class="sig-date-seg"></span><span>/</span><span class="sig-date-seg"></span></div>
		</div>
		<div class="sig">
			<div class="sig-t">ผู้ตรวจรับ / Inspected By</div>
			<div class="sig-write"><span class="sig-write-lbl">ลงชื่อ</span><span class="sig-ln"></span></div>
			<div class="sig-date-row"><span>วันที่</span><span class="sig-date-seg"></span><span>/</span><span class="sig-date-seg"></span><span>/</span><span class="sig-date-seg"></span></div>
			<div class="sig-sub">%s</div>
		</div>
	</div>
</body>
</html>`, html.EscapeString(receipt.DocumentNo), html.EscapeString(receipt.DocumentNo), html.EscapeString(string(receipt.Status)), html.EscapeString(receipt.ReceivedAt.In(time.Local).Format("02 Jan 2006 15:04")), html.EscapeString(receipt.WarehouseName), html.EscapeString(receipt.SupplierName), html.EscapeString(receipt.PurchaseOrderNo), html.EscapeString(receipt.ReferenceNo), html.EscapeString(receipt.Note), rows.String(), receipt.SubtotalAmount, receipt.DiscountAmount, receipt.NetAmount, receipt.VATPercent, receipt.VATAmount, receipt.TotalAmount, html.EscapeString(receipt.ConfirmedByName))
}
