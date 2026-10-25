-- Additional demo catalogue. Prices are sample daily rates in kopecks.
-- Safe to rerun: existing models, prices and equipment are not overwritten.
BEGIN;
CREATE TEMP TABLE extra_tools (
    id text, name text, category text, description text, price bigint, quantity integer
) ON COMMIT DROP;
INSERT INTO extra_tools VALUES
 ('extra-cordless-driver','Аккумуляторный шуруповёрт 18 В','drills','Для сборки мебели, монтажа и работы с крепежом. Аккумулятор и зарядное устройство в комплекте.',70000,4),
 ('extra-impact-drill','Ударная дрель','drills','Сверление дерева, металла и кирпича. Боковая рукоятка в комплекте.',60000,3),
 ('extra-angle-grinder','Угловая шлифмашина 125 мм','grinders','Для резки и зачистки металла. Расходные диски приобретаются отдельно.',65000,4),
 ('extra-belt-sander','Ленточная шлифмашина','grinders','Для обработки деревянных поверхностей, дверей и столешниц.',90000,2),
 ('extra-orbital-sander','Эксцентриковая шлифмашина','grinders','Для финишной обработки дерева и удаления старого покрытия.',75000,3),
 ('extra-circular-saw','Дисковая пила','saws','Прямой распил досок, фанеры и листовых материалов.',100000,3),
 ('extra-reciprocating-saw','Сабельная пила','saws','Для демонтажа и распила в труднодоступных местах. Полотно подбирается под материал.',95000,2),
 ('extra-mitre-saw','Торцовочная пила','saws','Точный поперечный распил и рез под углом. Для плинтусов, наличников и бруса.',140000,2),
 ('extra-dust-extractor','Строительный пылесос','vacuums','Сбор строительной пыли и подключение к электроинструменту.',110000,3),
 ('extra-laser-level','Лазерный уровень с треногой','levels','Разметка при укладке плитки, монтаже перегородок и установке мебели.',80000,3),
 ('extra-welding-inverter','Сварочный инвертор','welders','Для ручной дуговой сварки. Кабели в комплекте; электроды и средства защиты отдельно.',130000,2),
 ('extra-demolition-hammer','Отбойный молоток','rotary-hammers','Для демонтажа бетонных конструкций и удаления старой стяжки.',190000,2);
INSERT INTO models(id,name,category,description,daily_price)
 SELECT id,name,category,description,price FROM extra_tools
 ON CONFLICT(id) DO NOTHING;
INSERT INTO units(id,model_id,state)
 SELECT md5(t.id || '-unit-' || n),t.id,'ready'
 FROM extra_tools t CROSS JOIN LATERAL generate_series(1,t.quantity) AS n
 ON CONFLICT(id) DO NOTHING;
COMMIT;
