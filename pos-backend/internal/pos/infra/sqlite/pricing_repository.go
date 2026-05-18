package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"pos-backend/internal/pos/domain"
	"pos-backend/internal/pos/domain/pricing"
)

func (r *Repository) CreateOrderDiscount(ctx context.Context, v *pricing.OrderDiscount) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO order_line_discounts(id,order_id,order_line_id,pricing_policy_id,scope,application_index,amount_kind,amount_minor,value_basis_points,reason,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.OrderID, nullableString(v.OrderLineID), nullableString(v.PricingPolicyID), string(v.Scope), v.ApplicationIndex, string(v.AmountKind), v.AmountMinor, v.ValueBasisPoints, nullableString(v.Reason), dbTime(v.CreatedAt))
	return normalizeErr(err)
}

func (r *Repository) CreateOrderSurcharge(ctx context.Context, v *pricing.OrderSurcharge) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO order_surcharges(id,order_id,pricing_policy_id,kind,application_index,amount_kind,amount_minor,value_basis_points,reason,created_at) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.OrderID, nullableString(v.PricingPolicyID), string(v.Kind), v.ApplicationIndex, string(v.AmountKind), v.AmountMinor, v.ValueBasisPoints, nullableString(v.Reason), dbTime(v.CreatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListOrderDiscounts(ctx context.Context, orderID string) ([]pricing.OrderDiscount, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,order_id,order_line_id,pricing_policy_id,scope,application_index,amount_kind,amount_minor,value_basis_points,reason,created_at FROM order_line_discounts WHERE order_id = ? ORDER BY application_index, id`, orderID)
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer rows.Close()
	var out []pricing.OrderDiscount
	for rows.Next() {
		v, err := scanOrderDiscount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, normalizeErr(rows.Err())
}

func (r *Repository) ListOrderSurcharges(ctx context.Context, orderID string) ([]pricing.OrderSurcharge, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,order_id,pricing_policy_id,kind,application_index,amount_kind,amount_minor,value_basis_points,reason,created_at FROM order_surcharges WHERE order_id = ? ORDER BY application_index, id`, orderID)
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer rows.Close()
	var out []pricing.OrderSurcharge
	for rows.Next() {
		v, err := scanOrderSurcharge(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, normalizeErr(rows.Err())
}

func (r *Repository) ListActivePricingPolicies(ctx context.Context, restaurantID string) ([]pricing.PricingPolicy, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,restaurant_id,kind,name,scope,amount_kind,amount_minor,value_basis_points,application_index,requires_permission,manual,active,created_at,updated_at FROM pricing_policies WHERE restaurant_id = ? AND active = 1 ORDER BY application_index, id`, restaurantID)
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer rows.Close()
	var out []pricing.PricingPolicy
	for rows.Next() {
		var v pricing.PricingPolicy
		var kind, scope, amountKind, created, updated string
		var active, manual int
		if err := rows.Scan(&v.ID, &v.RestaurantID, &kind, &v.Name, &scope, &amountKind, &v.AmountMinor, &v.ValueBasisPoints, &v.ApplicationIndex, &v.RequiresPermission, &manual, &active, &created, &updated); err != nil {
			return nil, normalizeErr(err)
		}
		v.Kind = pricing.PricingPolicyKind(kind)
		v.Scope = pricing.DiscountScope(scope)
		v.AmountKind = pricing.AmountKind(amountKind)
		v.Manual = manual == 1
		v.Active = active == 1
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, normalizeErr(rows.Err())
}

func (r *Repository) GetPricingPolicy(ctx context.Context, id string) (*pricing.PricingPolicy, error) {
	row := r.queryer(ctx).QueryRowContext(ctx, `SELECT id,restaurant_id,kind,name,scope,amount_kind,amount_minor,value_basis_points,application_index,requires_permission,manual,active,created_at,updated_at FROM pricing_policies WHERE id = ?`, id)
	var v pricing.PricingPolicy
	var kind, scope, amountKind, created, updated string
	var active, manual int
	if err := row.Scan(&v.ID, &v.RestaurantID, &kind, &v.Name, &scope, &amountKind, &v.AmountMinor, &v.ValueBasisPoints, &v.ApplicationIndex, &v.RequiresPermission, &manual, &active, &created, &updated); err != nil {
		return nil, normalizeErr(err)
	}
	v.Kind = pricing.PricingPolicyKind(kind)
	v.Scope = pricing.DiscountScope(scope)
	v.AmountKind = pricing.AmountKind(amountKind)
	v.Manual = manual == 1
	v.Active = active == 1
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

func (r *Repository) ListTaxProfilesByIDs(ctx context.Context, ids []string) (map[string]pricing.TaxProfile, error) {
	out := map[string]pricing.TaxProfile{}
	for _, id := range ids {
		var v pricing.TaxProfile
		var taxExempt, active int
		var created, updated string
		err := r.queryer(ctx).QueryRowContext(ctx, `SELECT id,name,tax_exempt,active,created_at,updated_at FROM tax_profiles WHERE id = ?`, id).
			Scan(&v.ID, &v.Name, &taxExempt, &active, &created, &updated)
		if err != nil {
			err = normalizeErr(err)
			if errors.Is(err, domain.ErrNotFound) {
				continue
			}
			return nil, err
		}
		v.TaxExempt = taxExempt == 1
		v.Active = active == 1
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out[v.ID] = v
	}
	return out, nil
}

func (r *Repository) ListTaxRulesByProfileIDs(ctx context.Context, ids []string) (map[string][]pricing.TaxRule, error) {
	out := map[string][]pricing.TaxRule{}
	for _, id := range ids {
		rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,tax_profile_id,name,kind,mode,rate_basis_points,amount_minor,compound,priority,active,created_at,updated_at FROM tax_rules WHERE tax_profile_id = ? ORDER BY priority, id`, id)
		if err != nil {
			return nil, normalizeErr(err)
		}
		for rows.Next() {
			var v pricing.TaxRule
			var kind, mode, created, updated string
			var compound, active int
			if err := rows.Scan(&v.ID, &v.TaxProfileID, &v.Name, &kind, &mode, &v.RateBasisPoints, &v.AmountMinor, &compound, &v.Priority, &active, &created, &updated); err != nil {
				_ = rows.Close()
				return nil, normalizeErr(err)
			}
			v.Kind = pricing.TaxRuleKind(kind)
			v.Mode = pricing.TaxMode(mode)
			v.Compound = compound == 1
			v.Active = active == 1
			v.CreatedAt = parseTime(created)
			v.UpdatedAt = parseTime(updated)
			out[v.TaxProfileID] = append(out[v.TaxProfileID], v)
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, normalizeErr(err)
		}
		if err := rows.Close(); err != nil {
			return nil, normalizeErr(err)
		}
	}
	return out, nil
}

func (r *Repository) CreatePrecheckBreakdown(ctx context.Context, precheckID string, result pricing.CalculationResult) error {
	for _, line := range result.Lines {
		_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO precheck_lines(precheck_id,order_line_id,menu_item_id,catalog_item_id,name,quantity,unit_price_minor,subtotal_minor,discount_total_minor,surcharge_total_minor,taxable_base_minor,tax_total_minor,tax_added_minor,total_minor,currency_code,tax_profile_id) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			precheckID, line.OrderLineID, line.MenuItemID, line.CatalogItemID, line.Name, line.Quantity, line.UnitPriceMinor, line.SubtotalMinor, line.DiscountTotalMinor, line.SurchargeTotalMinor, line.TaxableBaseMinor, line.TaxTotalMinor, line.TaxAddedMinor, line.TotalMinor, line.CurrencyCode, nullableString(line.TaxProfileID))
		if err != nil {
			return normalizeErr(err)
		}
		for _, modifier := range line.Modifiers {
			_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO precheck_line_modifiers(precheck_id,order_line_id,modifier_group_id,modifier_option_id,name,quantity,unit_price_minor,total_minor) VALUES (?,?,?,?,?,?,?,?)`,
				precheckID, line.OrderLineID, modifier.ModifierGroupID, modifier.ModifierOptionID, modifier.Name, modifier.Quantity, modifier.UnitPriceMinor, modifier.TotalMinor)
			if err != nil {
				return normalizeErr(err)
			}
		}
	}
	for _, discount := range result.Discounts {
		_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO precheck_discounts(precheck_id,discount_id,pricing_policy_id,scope,application_index,order_line_id,amount_kind,amount_minor,reason) VALUES (?,?,?,?,?,?,?,?,?)`,
			precheckID, discount.DiscountID, nullableString(discount.PricingPolicyID), string(discount.Scope), discount.ApplicationIndex, nullableString(discount.OrderLineID), string(discount.AmountKind), discount.AmountMinor, nullableString(discount.Reason))
		if err != nil {
			return normalizeErr(err)
		}
	}
	for _, surcharge := range result.Surcharges {
		_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO precheck_surcharges(precheck_id,surcharge_id,pricing_policy_id,kind,application_index,amount_kind,amount_minor,reason) VALUES (?,?,?,?,?,?,?,?)`,
			precheckID, surcharge.SurchargeID, nullableString(surcharge.PricingPolicyID), string(surcharge.Kind), surcharge.ApplicationIndex, string(surcharge.AmountKind), surcharge.AmountMinor, nullableString(surcharge.Reason))
		if err != nil {
			return normalizeErr(err)
		}
	}
	for _, tax := range result.Taxes {
		_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO precheck_taxes(precheck_id,order_line_id,tax_profile_id,tax_rule_id,name,kind,mode,rate_basis_points,taxable_base_minor,tax_amount_minor,compound,priority) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
			precheckID, tax.OrderLineID, tax.TaxProfileID, tax.TaxRuleID, tax.Name, string(tax.Kind), string(tax.Mode), tax.RateBasisPoints, tax.TaxableBaseMinor, tax.TaxAmountMinor, boolInt(tax.Compound), tax.Priority)
		if err != nil {
			return normalizeErr(err)
		}
	}
	return nil
}

type orderDiscountScanner interface {
	Scan(...any) error
}

func scanOrderDiscount(row orderDiscountScanner) (*pricing.OrderDiscount, error) {
	var v pricing.OrderDiscount
	var lineID, policyID, reason sql.NullString
	var scope, amountKind, created string
	if err := row.Scan(&v.ID, &v.OrderID, &lineID, &policyID, &scope, &v.ApplicationIndex, &amountKind, &v.AmountMinor, &v.ValueBasisPoints, &reason, &created); err != nil {
		return nil, normalizeErr(err)
	}
	v.OrderLineID = stringPtr(lineID)
	v.PricingPolicyID = stringPtr(policyID)
	v.Scope = pricing.DiscountScope(scope)
	v.AmountKind = pricing.AmountKind(amountKind)
	v.Reason = stringPtr(reason)
	v.CreatedAt = parseTime(created)
	return &v, nil
}

func scanOrderSurcharge(row orderDiscountScanner) (*pricing.OrderSurcharge, error) {
	var v pricing.OrderSurcharge
	var kind, amountKind, created string
	var policyID, reason sql.NullString
	if err := row.Scan(&v.ID, &v.OrderID, &policyID, &kind, &v.ApplicationIndex, &amountKind, &v.AmountMinor, &v.ValueBasisPoints, &reason, &created); err != nil {
		return nil, normalizeErr(err)
	}
	v.PricingPolicyID = stringPtr(policyID)
	v.Kind = pricing.SurchargeKind(kind)
	v.AmountKind = pricing.AmountKind(amountKind)
	v.Reason = stringPtr(reason)
	v.CreatedAt = parseTime(created)
	return &v, nil
}
