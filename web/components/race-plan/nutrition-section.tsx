import { create } from "@bufbuild/protobuf";
import { TextAreaField, TextField } from "@components/ui/fields";
import { Stat } from "@components/ui/misc";
import {
  inputClass,
  tableClass,
  tdClass,
  thClass,
} from "@components/ui/styles";
import type {
  NutritionProduct,
  RacePlanAnalysis,
} from "gen/races/v1/race_plans_pb";
import {
  NutritionProductSchema,
  NutritionStrategySchema,
} from "gen/races/v1/race_plans_pb";
import { numberInput, parseNumber } from "helpers/parse";
import React, { useState } from "react";

import ListControls, { AddButton } from "./list-controls";
import type { SectionComponentProps } from "./section";
import Section, {
  EmptyText,
  TextBlock,
  nextKey,
  useSectionEdit,
} from "./section";

interface ProductDraft {
  key: number;
  name: string;
  carbs: string;
  sodium: string;
  fluid: string;
  caffeine: string;
}

const toDraft = (product: NutritionProduct): ProductDraft => ({
  caffeine: numberInput(product.caffeineMg),
  carbs: numberInput(product.carbsGrams),
  fluid: numberInput(product.fluidMl),
  key: nextKey(),
  name: product.name,
  sodium: numberInput(product.sodiumMg),
});

interface NutritionSectionProps extends SectionComponentProps {
  analysis: RacePlanAnalysis | undefined;
}

const NutritionSection: React.FC<NutritionSectionProps> = ({
  plan,
  analysis,
  rules,
  onSave,
}) => {
  const edit = useSectionEdit(onSave);
  const { nutrition } = plan;
  const [carbs, setCarbs] = useState("");
  const [fluid, setFluid] = useState("");
  const [sodium, setSodium] = useState("");
  const [notes, setNotes] = useState("");
  const [products, setProducts] = useState<ProductDraft[]>([]);

  const startEdit = () => {
    setCarbs(numberInput(nutrition?.carbsGramsPerHour ?? 0));
    setFluid(numberInput(nutrition?.fluidMlPerHour ?? 0));
    setSodium(numberInput(nutrition?.sodiumMgPerHour ?? 0));
    setNotes(nutrition?.notes ?? "");
    setProducts((nutrition?.products ?? []).map((product) => toDraft(product)));
    edit.start();
  };

  const submit = () => {
    const carbsValue = parseNumber(carbs);
    const fluidValue = parseNumber(fluid);
    const sodiumValue = parseNumber(sodium);
    if (carbsValue === null || fluidValue === null || sodiumValue === null) {
      edit.setValidation("Hourly targets must be positive numbers.");
      return;
    }
    const parsed: NutritionProduct[] = [];
    for (const product of products) {
      const productCarbs = parseNumber(product.carbs);
      const productSodium = parseNumber(product.sodium);
      const productFluid = parseNumber(product.fluid);
      const productCaffeine = parseNumber(product.caffeine);
      if (
        productCarbs === null ||
        productSodium === null ||
        productFluid === null ||
        productCaffeine === null
      ) {
        edit.setValidation(
          `${product.name === "" ? "A product" : product.name}: amounts must be positive numbers.`
        );
        return;
      }
      if (product.name.trim() === "") {
        continue;
      }
      parsed.push(
        create(NutritionProductSchema, {
          caffeineMg: productCaffeine,
          carbsGrams: productCarbs,
          fluidMl: productFluid,
          name: product.name.trim(),
          sodiumMg: productSodium,
        })
      );
    }
    edit.save(
      {
        ...plan,
        nutrition: create(NutritionStrategySchema, {
          carbsGramsPerHour: carbsValue,
          fluidMlPerHour: fluidValue,
          notes,
          products: parsed,
          sodiumMgPerHour: sodiumValue,
        }),
      },
      "nutrition"
    );
  };

  const updateProduct = (changed: ProductDraft) => {
    setProducts(
      products.map((product) =>
        product.key === changed.key ? changed : product
      )
    );
  };

  if (edit.editing) {
    return (
      <Section
        section="nutrition"
        title="Nutrition"
        rules={rules}
        editable
        edit={edit}
        onStartEdit={startEdit}
        onSubmit={submit}
      >
        <div className="grid grid-cols-3 gap-3">
          <TextField
            label="Carbs (g/h)"
            value={carbs}
            onChange={setCarbs}
            inputMode="numeric"
            placeholder="90"
          />
          <TextField
            label="Fluid (ml/h)"
            value={fluid}
            onChange={setFluid}
            inputMode="numeric"
            placeholder="500"
          />
          <TextField
            label="Sodium (mg/h)"
            value={sodium}
            onChange={setSodium}
            inputMode="numeric"
            placeholder="600"
          />
        </div>
        <div className="flex flex-col gap-2">
          <span className="text-xs font-medium text-gray-600">
            Products (per serving)
          </span>
          <div className="-mx-4 overflow-x-auto px-4">
            <table className={tableClass}>
              <thead>
                <tr>
                  <th className={thClass}>Name</th>
                  <th className={thClass}>Carbs g</th>
                  <th className={thClass}>Sodium mg</th>
                  <th className={thClass}>Fluid ml</th>
                  <th className={thClass}>Caffeine mg</th>
                  <th className={thClass} />
                </tr>
              </thead>
              <tbody>
                {products.map((product, index) => (
                  <tr key={product.key}>
                    <td className="py-1 pr-1">
                      <input
                        value={product.name}
                        className={`${inputClass} min-w-40`}
                        aria-label="Product name"
                        placeholder="Maurten Gel 100"
                        onChange={(event) => {
                          updateProduct({
                            ...product,
                            name: event.target.value,
                          });
                        }}
                      />
                    </td>
                    <td className="py-1 pr-1">
                      <input
                        value={product.carbs}
                        inputMode="decimal"
                        className={`${inputClass} w-20`}
                        aria-label="Carbs"
                        onChange={(event) => {
                          updateProduct({
                            ...product,
                            carbs: event.target.value,
                          });
                        }}
                      />
                    </td>
                    <td className="py-1 pr-1">
                      <input
                        value={product.sodium}
                        inputMode="decimal"
                        className={`${inputClass} w-20`}
                        aria-label="Sodium"
                        onChange={(event) => {
                          updateProduct({
                            ...product,
                            sodium: event.target.value,
                          });
                        }}
                      />
                    </td>
                    <td className="py-1 pr-1">
                      <input
                        value={product.fluid}
                        inputMode="decimal"
                        className={`${inputClass} w-20`}
                        aria-label="Fluid"
                        onChange={(event) => {
                          updateProduct({
                            ...product,
                            fluid: event.target.value,
                          });
                        }}
                      />
                    </td>
                    <td className="py-1 pr-1">
                      <input
                        value={product.caffeine}
                        inputMode="decimal"
                        className={`${inputClass} w-20`}
                        aria-label="Caffeine"
                        onChange={(event) => {
                          updateProduct({
                            ...product,
                            caffeine: event.target.value,
                          });
                        }}
                      />
                    </td>
                    <td className="py-1">
                      <ListControls
                        index={index}
                        count={products.length}
                        onRemove={(position) => {
                          setProducts(
                            products.filter(
                              (_product, other) => other !== position
                            )
                          );
                        }}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <AddButton
            label="Add product"
            onClick={() => {
              setProducts([
                ...products,
                toDraft(create(NutritionProductSchema)),
              ]);
            }}
          />
        </div>
        <TextAreaField
          label="Notes"
          value={notes}
          onChange={setNotes}
          placeholder="Caffeine strategy, what to eat at aid stations"
        />
      </Section>
    );
  }

  return (
    <Section
      section="nutrition"
      title="Nutrition"
      rules={rules}
      editable={onSave !== undefined}
      edit={edit}
      onStartEdit={startEdit}
      onSubmit={submit}
    >
      <div className="flex flex-col gap-4">
        <div className="grid grid-cols-3 gap-3">
          <Stat
            label="Carbs"
            value={`${Math.round(nutrition?.carbsGramsPerHour ?? 0)} g/h`}
          />
          <Stat
            label="Fluid"
            value={`${Math.round(nutrition?.fluidMlPerHour ?? 0)} ml/h`}
          />
          <Stat
            label="Sodium"
            value={`${Math.round(nutrition?.sodiumMgPerHour ?? 0)} mg/h`}
          />
        </div>
        {analysis !== undefined && analysis.totalCarbsGrams > 0 && (
          <p className="text-sm text-gray-700">
            Planned in total: {Math.round(analysis.totalCarbsGrams)} g carbs,{" "}
            {(analysis.totalFluidMl / 1000).toFixed(1)} l fluid,{" "}
            {Math.round(analysis.totalSodiumMg)} mg sodium
            {analysis.totalCaffeineMg > 0 &&
              `, ${Math.round(analysis.totalCaffeineMg)} mg caffeine`}
            .
          </p>
        )}
        {(nutrition?.products ?? []).length === 0 ? (
          <EmptyText>No products yet.</EmptyText>
        ) : (
          <div className="-mx-4 overflow-x-auto px-4">
            <table className={tableClass}>
              <thead>
                <tr>
                  <th className={thClass}>Product</th>
                  <th className={`${thClass} text-right`}>Carbs g</th>
                  <th className={`${thClass} text-right`}>Sodium mg</th>
                  <th className={`${thClass} text-right`}>Fluid ml</th>
                  <th className={`${thClass} text-right`}>Caffeine mg</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {(nutrition?.products ?? []).map((product) => (
                  <tr key={product.name}>
                    <td className={`${tdClass} font-medium`}>{product.name}</td>
                    <td className={`${tdClass} text-right tabular-nums`}>
                      {product.carbsGrams === 0
                        ? "–"
                        : numberInput(product.carbsGrams)}
                    </td>
                    <td className={`${tdClass} text-right tabular-nums`}>
                      {product.sodiumMg === 0
                        ? "–"
                        : numberInput(product.sodiumMg)}
                    </td>
                    <td className={`${tdClass} text-right tabular-nums`}>
                      {product.fluidMl === 0
                        ? "–"
                        : numberInput(product.fluidMl)}
                    </td>
                    <td className={`${tdClass} text-right tabular-nums`}>
                      {product.caffeineMg === 0
                        ? "–"
                        : numberInput(product.caffeineMg)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        <TextBlock label="Notes" text={nutrition?.notes ?? ""} />
      </div>
    </Section>
  );
};

export default NutritionSection;
