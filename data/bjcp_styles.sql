INSERT INTO beer_category_bjcps(name, id)
VALUES
    ('Standard American Beer', 1),
    ('International Lager', 2),
    ('Czech Lager', 3),
    ('Pale Malty European Lager', 4),
    ('Pale Bitter European Beer', 5),
    ('Amber Malty European Lager', 6),
    ('Amber Bitter European Beer', 7),
    ('Dark European Lager', 8),
    ('Strong European Beer', 9),
    ('German Wheat Beer', 10),
    ('British Bitter', 11),
    ('Pale Commonwealth Beer', 12),
    ('Brown British Beer', 13),
    ('Scottish Ale', 14),
    ('Irish Beer', 15),
    ('Dark British Beer', 16),
    ('Strong British Ale', 17),
    ('Pale American Ale', 18),
    ('Amber and Brown American Beer', 19),
    ('American Porter and Stout', 20),
    ('IPA', 21),
    ('Strong American Ale', 22),
    ('European Sour Ale', 23),
    ('Belgian Ale', 24),
    ('Strong Belgian Ale', 25),
    ('Monastic Ale', 26),
    ('Historical Ale', 27),
    ('American Wild Ale', 28),
    ('Fruit Beer', 29),
    ('Spiced Beer', 30),
    ('Alternative Fermentables Beer', 31),
    ('Smoked Beer', 32),
    ('Wood Beer', 33),
    ('Specialty Beer', 34),
    ('Traditional Cider', 101),
    ('Strong Cider', 102),
    ('Specialty Cider', 103),
    ('Perry', 104),
    ('Traditional Mead', 1001),
    ('Fruit Mead', 1002),
    ('Spiced Mead', 1003),
    ('Specialty Mead', 1004),
    ('Other', 9999)
ON CONFLICT DO NOTHING;

INSERT INTO beer_style_families(id, name)
VALUES
    (1,'Amber/Red Ale'),
    (2, 'Amber/Red Lager'),
    (3, 'Bock'),
    (4, 'Brown Ale'),
    (5, 'Dark Ale'),
    (6, 'Dark Lager'),
    (7, 'Farmhouse Ale'),
    (8, 'Hybrid/Specialty Beer'),
    (9, 'IPA'),
    (10, 'Pale/Blonde Ale'),
    (11, 'Pale Lager'),
    (12, 'Pilsner'),
    (13, 'Porter'),
    (14, 'Stout'),
    (15, 'Strong Ale'),
    (16, 'Wheat Beer'),
    (17, 'Wild/Sour Beer'),
    (18, 'Cider'),
    (19, 'Mead'),
    (9999, 'Other')
ON CONFLICT DO NOTHING;

INSERT INTO beer_style_bjcps(bjcp_id, name, category_id, family_id)
VALUES
    ('1A', 'American Light Lager', 1, 11),
    ('1B', 'American Lager', 1, 11),
    ('1C', 'Cream Ale', 1, 8),
    ('2A', 'International Pale Lager', 2, 11),
    ('2B', 'International Amber Lager', 2, 2),
    ('2C', 'International Dark Lager', 2, 6),
    ('3A', 'Czech Pale Lager', 3, 12),
    ('3B', 'Czech Premium Pale Lager', 3, 12),
    ('3C', 'Czech Amber Lager', 3, 2),
    ('3D', 'Czech Dark Lager', 3, 6),
    ('4A', 'Munich Helles', 4, 11),
    ('4B', 'Festbier', 4, 11),
    ('4C', 'Helles Bock', 4, 3),
    ('5A', 'German Leichtbier', 5, 12),
    ('5B', 'Kölsch', 5, 10),
    ('5C', 'German Helles Exportbier', 5, 11),
    ('5D', 'German Pils', 5, 12),
    ('6A', 'Märzen', 6, 2),
    ('6B', 'Rauchbier', 6, 8),
    ('6C', 'Dunkles Bock', 6, 3),
    ('7A', 'Vienna Lager', 7, 2),
    ('7B', 'Altbier', 7, 4),
    ('8A', 'Munich Dunkel', 8, 6),
    ('8B', 'Schwarzbier', 8, 6),
    ('9A', 'Doppelbock', 9, 3),
    ('9B', 'Eisbock', 9, 3),
    ('9C', 'Baltic Porter', 9, 13),
    ('10A', 'Weissbier', 10, 16),
    ('10B', 'Dunkles Weissen', 10, 16),
    ('10C', 'Weizenbock', 10, 3),
    ('11A', 'Ordinary Bitter', 11, 10),
    ('11B', 'Best Bitter', 11, 10),
    ('11C', 'Strong Bitter', 11, 15),
    ('12A', 'British Golden Ale', 12, 10),
    ('12B', 'Australian Sparkling Ale', 12, 10),
    ('12C', 'English IPA', 12, 9),
    ('13A', 'Dark Mild', 13, 5),
    ('13B', 'British Brown Ale', 13, 4),
    ('13C', 'English Porter', 13, 13),
    ('14A', 'Scottish Light', 14, 5),
    ('14B', 'Scottish Heavy', 14, 15),
    ('14C', 'Scottish Export', 14, 5),
    ('15A', 'Irish Red Ale', 15, 1),
    ('15B', 'Irish Stout', 15, 14),
    ('15C', 'Irish Extra Stout', 15, 14),
    ('16A', 'Sweet Stout', 16, 14),
    ('16B', 'Oatmeal Stout', 16, 14),
    ('16C', 'Tropical Stout', 16, 14),
    ('16D', 'Foreign Extra Stout', 16, 14),
    ('17A', 'British Strong Ale', 17, 15),
    ('17B', 'Old Ale', 17, 15),
    ('17C', 'Wee Heavy', 17, 15),
    ('17D', 'English Barley Wine', 17, 15),
    ('18A', 'Blonde Ale', 18, 10),
    ('18B', 'American Pale Ale', 18, 10),
    ('19A', 'American Amber Ale', 19, 1),
    ('19B', 'California Common', 19, 8),
    ('19C', 'American Brown Ale', 19, 4),
    ('20A', 'American Porter', 20, 13),
    ('20B', 'American Stout', 20, 14),
    ('20C', 'Imperial Stout', 20, 14),
    ('21A', 'American IPA', 21, 9),
    ('21B', 'Speciality IPA', 21, 9),
    ('21C', 'Hazy IPA', 21, 9),
    ('22A', 'Double IPA', 22, 9),
    ('22B', 'American Strong Ale', 22, 15),
    ('22C', 'American Barleywine', 22, 15),
    ('22D', 'Wheatwine', 22, 15),
    ('23A', 'Berliner Weisse', 23, 17),
    ('23B', 'Flanders Red Ale', 23, 17),
    ('23C', 'Oud Bruin', 23, 17),
    ('23D', 'Lambic', 23, 17),
    ('23E', 'Gueuze', 23, 17),
    ('23F', 'Fruit Lambic', 23, 17),
    ('23G', 'Gose', 23, 17),
    ('24A', 'Witbier', 24, 16),
    ('24B', 'Belgian Pale Ale', 24, 10),
    ('24C', 'Bière de Garde', 24, 7),
    ('25A', 'Belgian Blonde Ale', 25, 10),
    ('25B', 'Saison', 25, 7),
    ('25C', 'Belgian Golden Strong Ale', 25, 15),
    ('26A', 'Belgian Single', 26, 10),
    ('26B', 'Belgian Dubbel', 26, 1),
    ('26C', 'Belgian Tripel', 26, 15),
    ('26D', 'Belgian Dark Strong Ale', 26, 15),
    ('27A', 'Historical Beer: Adambier', 27, 8),
    ('27B', 'Historical Beer: Kellerbier', 27, 11),
    ('27C', 'Historical Beer: Kentucky Common', 27, 1),
    ('27D', 'Historical Beer: Kvass', 27,  8),
    ('27E', 'Historical Beer: Lichtenhainer', 27, 16),
    ('27F', 'Historical Beer: London Brown Ale', 27, 4),
    ('27G', 'Historical Beer: Piwo Grodziskie', 27, 16),
    ('27H', 'Historical Beer: Pre-Prohibition Lager', 27, 4),
    ('27I', 'Historical Beer: Pre-Prohibition Porter', 27, 13),
    ('27J', 'Historical Beer: Roggenbier', 27, 8),
    ('27K', 'Historical Beer: Sahti', 27, 7),
    ('28A', 'Brett Beer', 28,17),
    ('28B', 'Mixed-Fermentation Sour Beer', 28, 17),
    ('28C', 'Wild Specialty Beer', 28, 17),
    ('28D', 'Straight Sour Beer', 28, 17),
    ('29A', 'Fruit Beer', 29, 8),
    ('29B', 'Fruit and Spice Beer', 29, 8),
    ('29C', 'Specialty Fruit Beer', 29, 8),
    ('29D', 'Grape Ale', 29, 8),
    ('30A', 'Spice, Herb or Vegetable Beer', 30, 8),
    ('30B', 'Autumn Seasonal Beer', 30, 8),
    ('30C', 'Winter Seasonal Beer', 30, 8),
    ('30D', 'Specialty Spice Beer', 30, 8),
    ('31A', 'Alternative Grain Beer', 31, 8),
    ('31B', 'Alternative Sugar Beer', 31, 8),
    ('32A', 'Classic Style Smoked Beer', 32, 8),
    ('32B', 'Specialty Smoked Beer', 32, 8),
    ('33A', 'Wood-Aged Beer', 33, 8),
    ('33B', 'Specialty Wood-Aged Beer', 33, 8),
    ('34A', 'Commercial Specialty Beer', 34, 8),
    ('34B', 'Mixed-Style Beer', 34, 8),
    ('34C', 'Experimental Beer', 34, 8),
    ('C1A', 'Common Cider', 101, 18),
    ('C1B', 'Heirloom Cider', 101, 18),
    ('C1C', 'English Cider', 101, 18),
    ('C1D', 'French Cider', 101, 18),
    ('C1E', 'Spanish Cider', 101, 18),
    ('C2A', 'New England Cider', 102, 18),
    ('C2B', 'Applewine', 102, 18),
    ('C2C', 'Ice Cider', 102, 18),
    ('C2D', 'Fire Cider', 102, 18),
    ('C3A', 'Fruit Cider', 103, 18),
    ('C3B', 'Spiced Cider', 103, 18),
    ('C3C', 'Experimental Cider', 103, 18),
    ('C4A', 'Common Perry', 104, 18),
    ('C4B', 'Heirloom Perry', 104, 18),
    ('C4C', 'Ice Perry', 104, 18),
    ('C4D', 'Experimental Perry', 104, 18),
    ('M1A', 'Dry Mead', 1001, 19),
    ('M1B', 'Semi-Sweet Mead', 1001, 19),
    ('M1C', 'Sweet Mead', 1001, 19),
    ('M2A', 'Cyser', 1002, 19),
    ('M2B', 'Pyment', 1002, 19),
    ('M2C', 'Berry Mead', 1002, 19),
    ('M2D', 'Stone Fruit Mead', 1002, 19),
    ('M2E', 'Melomel', 1002, 19),
    ('M3A', 'Fruit and Spice Mead', 1003, 19),
    ('M3B', 'Spice, Herb or Vegetable Mead', 1003, 19),
    ('M4A', 'Braggot', 1004, 19),
    ('M4B', 'Historical Mead', 1004, 19),
    ('M4C', 'Experimental Mead', 1004, 19),
    ('OTHER', 'Other', 9999, 9999)
ON CONFLICT DO NOTHING;

INSERT INTO beer_styles(name, bjcp_style_id)
VALUES
    ('Altbier - Sticke' , '7B'),
    ('Altbier - Traditional' , '7B'),
    ('Australian Sparkling Ale' , '12B'),
    ('Belgian Blonde' , '25A'),
    ('Belgian Enkel / Patersbier' , '26A'),
    ('Bitter - Best' , '11B'),
    ('Bitter - Extra Special / Strong (ESB)' , '11C'),
    ('Bitter - Session / Ordinary' , '11A'),
    ('Bière de Champagne / Bière Brut' , '21B'),
    ('Blonde / Golden Ale - American' , '18A'),
    ('Blonde / Golden Ale - English' , '18A'),
    ('Blonde / Golden Ale - Other' , '18A'),
    ('Bock - Hell / Maibock / Lentebock' , '4C'),
    ('Bock - Single / Traditional' , '6C'),
    ('Bock - Weizenbock' , '10C'),
    ('Bock - Weizendoppelbock' , '9A'),
    ('Brett Beer' , '28A'),
    ('Brown Ale - American' , '19C'),
    ('Brown Ale - English' , '13B'),
    ('Brown Ale - Other' , '13B'),
    ('California Common' , '19B'),
    ('Cider - Applewine' , 'C2B'),
    ('Cider - Basque' , 'C1E'),
    ('Cider - Graff' , 'C2C'),
    ('Cider - Herbed / Spiced / Hopped' , 'C3B'),
    ('Cider - Ice' , 'C2C'),
    ('Cider - Other Fruit' , 'C3A'),
    ('Cider - Perry / Poiré' , 'C4A'),
    ('Cider - Rosé' , 'C3A'),
    ('Cider - Sweet' , 'C1A'),
    ('Cider - Traditional / Apfelwein' , 'C1B'),
    ('Corn Beer / Chicha de Jora' , '31A'),
    ('Cream Ale' , '1C'),
    ('Cream Ale - Imperial / Double' , '1C'),
    ('Farmhouse Ale - Bière de Coupage' , '33A'),
    ('Farmhouse Ale - Bière de Mars' , '24C'),
    ('Farmhouse Ale - Brett' , '28A'),
    ('Farmhouse Ale - Grisette' , '25B'),
    ('Farmhouse Ale - Kornøl' , '30A'),
    ('Farmhouse Ale - Sahti' , '27K'),
    ('Festbier' , '4B'),
    ('Flavored Malt Beverage' , '34B'),
    ('Gluten-Free' , '31A'),
    ('Golden Ale - Ukrainian' , '12A'),
    ('Grape Ale - Italian' , '29D'),
    ('Grape Ale - Other' , '29D'),
    ('Grodziskie / Grätzer' , '27G'),
    ('Happoshu' , '31A'),
    ('Historical Beer - Berliner Braunbier' , '10A'),
    ('Historical Beer - Broyhan' , '10A'),
    ('Historical Beer - Burton Ale' , '17A'),
    ('Historical Beer - Dampfbier' , '19B'),
    ('Historical Beer - Kentucky Common' , '27C'),
    ('Historical Beer - Kottbusser' , '31B'),
    ('Historical Beer - Kuit / Kuyt / Koyt' , '31A'),
    ('Historical Beer - Lichtenhainer' , '27E'),
    ('Historical Beer - Mumme' , '30A'),
    ('Historical Beer - Other' , '34B'),
    ('Historical Beer - Steinbier' , '17B'),
    ('Historical Beer - Zoigl' , '7A'),
    ('Honey Beer' , '31B'),
    ('IPA - Belgian' , '21B'),
    ('IPA - Brett' , '21B'),
    ('IPA - Brown' , '21B'),
    ('IPA - Brut' , '21B'),
    ('IPA - Cold' , '21B'),
    ('IPA - English' , '12C'),
    ('IPA - Farmhouse' , '21B'),
    ('IPA - Fruited' , '21B'),
    ('IPA - Imperial / Double Black' , '22A'),
    ('IPA - Imperial / Double Milkshake' , '22A'),
    ('IPA - Imperial / Double New England / Hazy' , '22A'),
    ('IPA - Milkshake' , '21B'),
    ('IPA - Other' , '21B'),
    ('IPA - Quadruple' , '21B'),
    ('IPA - Red' , '21B'),
    ('IPA - Rye' , '21B'),
    ('IPA - Session' , '21B'),
    ('IPA - Sour' , '21B'),
    ('IPA - Triple' , '22A'),
    ('IPA - Triple New England / Hazy' , '22A'),
    ('IPA - White / Wheat' , '21B'),
    ('Kellerbier / Zwickelbier' , '27B'),
    ('Koji / Ginjo Beer' , '34C'),
    ('Kvass' , '27D'),
    ('Kölsch' , '5B'),
    ('Lager - Amber / Red' , '2B'),
    ('Lager - American' , '1B'),
    ('Lager - American Amber / Red' , '19A'),
    ('Lager - American Light' , '1A'),
    ('Lager - American Pre-Prohibition' , '27H'),
    ('Lager - Dark' , '2C'),
    ('Lager - Dortmunder / Export' , '5C'),
    ('Lager - Helles' , '4A'),
    ('Lager - IPL (India Pale Lager)' , '34B'),
    ('Lager - Japanese Rice' , '2A'),
    ('Lager - Leichtbier' , '5A'),
    ('Lager - Mexican' , '2A'),
    ('Lager - Munich Dunkel' , '8A'),
    ('Lager - Other' , '2A'),
    ('Lager - Pale' , '2A'),
    ('Lager - Polotmavé (Czech Amber)' , '2B'),
    ('Lager - Rotbier' , '2B'),
    ('Lager - Smoked' , '32A'),
    ('Lager - Strong' , '9A'),
    ('Lager - Světlé (Czech Pale)' , '2A'),
    ('Lager - Tmavé (Czech Dark)' , '2C'),
    ('Lager - Vienna' , '7A'),
    ('Lager - Winter' , '30C'),
    ('Lambic - Faro' , '23D'),
    ('Malt Beer' , '19A'),
    ('Malt Liquor' , '19A'),
    ('Mead - Acerglyn / Maple Wine' , 'M4C'),
    ('Mead - Bochet' , 'M1A'),
    ('Mead - Braggot' , 'M4A'),
    ('Mead - Cyser' , 'M2A'),
    ('Mead - Melomel' , 'M2E'),
    ('Mead - Metheglin' , 'M3A'),
    ('Mead - Pyment' , 'M2B'),
    ('Mead - Session / Short' , 'M4C'),
    ('Mead - Traditional' , 'M4B'),
    ('Mild - Dark' , '13A'),
    ('Mild - Light' , '12A'),
    ('Mild - Other' , '13A'),
    ('Märzen' , '6A'),
    ('Pale Ale - American' , '18B'),
    ('Pale Ale - Australian' , '12B'),
    ('Pale Ale - English' , '11A'),
    ('Pale Ale - Fruited' , '29C'),
    ('Pale Ale - Milkshake' , '34B'),
    ('Pale Ale - New England / Hazy' , '18B'),
    ('Pale Ale - New Zealand' , '12B'),
    ('Pale Ale - Other' , '18B'),
    ('Pale Ale - XPA (Extra Pale)' , '18B'),
    ('Pilsner - Czech / Bohemian' , '3B'),
    ('Pilsner - German' , '5D'),
    ('Pilsner - Imperial / Double' , '2A'),
    ('Pilsner - Italian' , '2A'),
    ('Pilsner - New Zealand' , '2A'),
    ('Pilsner - Other' , '2A'),
    ('Porter - Baltic' , '9C'),
    ('Porter - English' , '13C'),
    ('Porter - Smoked' , '32A'),
    ('Rauchbier' , '6B'),
    ('Red Ale - American Amber / Red' , '19A'),
    ('Red Ale - Imperial / Double' , '19A'),
    ('Red Ale - Irish' , '15A'),
    ('Red Ale - Other' , '19A'),
    ('Roggenbier' , '27J'),
    ('Root Beer' , '30A'),
    ('Schwarzbier' , '8B'),
    ('Scottish Ale' , '14A'),
    ('Scottish Export Ale' , '14C'),
    ('Shandy / Radler' , '34B'),
    ('Sorghum / Millet Beer' , '31A'),
    ('Sour - Berliner Weisse' , '23A'),
    ('Sour - Catharina' , '23A'),
    ('Sour - Fruited Berliner Weisse' , '23A'),
    ('Sour - Other Gose' , '23G'),
    ('Sour - Smoothie / Pastry' , '34C'),
    ('Sour - Tomato / Vegetable Gose' , '23G'),
    ('Sour - Traditional Gose' , '23G'),
    ('Specialty Grain' , '31A'),
    ('Spiced / Herbed Beer' , '30A'),
    ('Stout - American' , '20B'),
    ('Stout - English' , '16D'),
    ('Stout - Imperial / Double White / Golden' , '20C'),
    ('Stout - White / Golden' , '20B'),
    ('Table Beer' , '24B'),
    ('Wheat Beer - American Pale Wheat' , '18B'),
    ('Wheat Beer - Dunkelweizen' , '10B'),
    ('Wheat Beer - Fruited' , '10A'),
    ('Wheat Beer - Hefeweizen' , '10A'),
    ('Wheat Beer - Hefeweizen Light / Leicht' , '10A'),
    ('Wheat Beer - Hopfenweisse' , '10A'),
    ('Wheat Beer - Kristallweizen' , '10A'),
    ('Wheat Beer - Other' , '10A'),
    ('Wheat Beer - Wheat Wine' , '22D'),
    ('Wheat Beer - Witbier / Blanche' , '24A'),
    ('Winter Warmer' , '30C'),
    ('Other', 'OTHER')
ON CONFLICT (name) DO UPDATE SET bjcp_style_id = excluded.bjcp_style_id;

UPDATE beer_styles SET bjcp_style_id='27A' WHERE name='Adambier';
UPDATE beer_styles SET bjcp_style_id='22C' WHERE name='Barleywine - American';
UPDATE beer_styles SET bjcp_style_id='17D' WHERE name='Barleywine - English';
UPDATE beer_styles SET bjcp_style_id='22C' WHERE name='Barleywine - Other';
UPDATE beer_styles SET bjcp_style_id='26B' WHERE name='Belgian Dubbel';
UPDATE beer_styles SET bjcp_style_id='26D' WHERE name='Belgian Quadrupel';
UPDATE beer_styles SET bjcp_style_id='26D' WHERE name='Belgian Strong Dark Ale';
UPDATE beer_styles SET bjcp_style_id='25C' WHERE name='Belgian Strong Golden Ale';
UPDATE beer_styles SET bjcp_style_id='26C' WHERE name='Belgian Tripel';
UPDATE beer_styles SET bjcp_style_id='9A' WHERE name='Bock - Doppelbock';
UPDATE beer_styles SET bjcp_style_id='9B' WHERE name='Bock - Eisbock';
UPDATE beer_styles SET bjcp_style_id='13B' WHERE name='Brown Ale - Belgian';
UPDATE beer_styles SET bjcp_style_id='19C' WHERE name='Brown Ale - Imperial / Double';
UPDATE beer_styles SET bjcp_style_id='30A' WHERE name='Chilli / Chile Beer';
UPDATE beer_styles SET bjcp_style_id='C1A' WHERE name='Cider - Dry';
UPDATE beer_styles SET bjcp_style_id='13A' WHERE name='Dark Ale';
UPDATE beer_styles SET bjcp_style_id='24C' WHERE name='Farmhouse Ale - Bière de Garde';
UPDATE beer_styles SET bjcp_style_id='25B' WHERE name='Farmhouse Ale - Other';
UPDATE beer_styles SET bjcp_style_id='25B' WHERE name='Farmhouse Ale - Saison';
UPDATE beer_styles SET bjcp_style_id='9B' WHERE name='Freeze-Distilled Beer';
UPDATE beer_styles SET bjcp_style_id='29A' WHERE name='Fruit Beer';
UPDATE beer_styles SET bjcp_style_id='30A' WHERE name='Hard Ginger Beer';
UPDATE beer_styles SET bjcp_style_id='27A' WHERE name='Historical Beer - Adambier';
UPDATE beer_styles SET bjcp_style_id='30A' WHERE name='Historical Beer - Gruit / Ancient Herbed Ale';
UPDATE beer_styles SET bjcp_style_id='21A' WHERE name='IPA - American';
UPDATE beer_styles SET bjcp_style_id='21B' WHERE name='IPA - Black / Cascadian Dark Ale';
UPDATE beer_styles SET bjcp_style_id='22A' WHERE name='IPA - Imperial / Double';
UPDATE beer_styles SET bjcp_style_id='21C' WHERE name='IPA - New England / Hazy';
UPDATE beer_styles SET bjcp_style_id='21C' WHERE name='IPA - New Zealand';
UPDATE beer_styles SET bjcp_style_id='23F' WHERE name='Lambic - Framboise';
UPDATE beer_styles SET bjcp_style_id='23F' WHERE name='Lambic - Fruit';
UPDATE beer_styles SET bjcp_style_id='23E' WHERE name='Lambic - Gueuze';
UPDATE beer_styles SET bjcp_style_id='23F' WHERE name='Lambic - Kriek';
UPDATE beer_styles SET bjcp_style_id='23D' WHERE name='Lambic - Other';
UPDATE beer_styles SET bjcp_style_id='23D' WHERE name='Lambic - Traditional';
UPDATE beer_styles SET bjcp_style_id='M4C' WHERE name='Mead - Other';
UPDATE beer_styles SET bjcp_style_id='17B' WHERE name='Old Ale';
UPDATE beer_styles SET bjcp_style_id='OTHER' WHERE name='Other';
UPDATE beer_styles SET bjcp_style_id='24B' WHERE name='Pale Ale - Belgian';
UPDATE beer_styles SET bjcp_style_id='20A' WHERE name='Porter - American';
UPDATE beer_styles SET bjcp_style_id='20A' WHERE name='Porter - Coffee';
UPDATE beer_styles SET bjcp_style_id='20A' WHERE name='Porter - Imperial / Double';
UPDATE beer_styles SET bjcp_style_id='9C' WHERE name='Porter - Imperial / Double Baltic';
UPDATE beer_styles SET bjcp_style_id='20A' WHERE name='Porter - Imperial / Double Coffee';
UPDATE beer_styles SET bjcp_style_id='20A' WHERE name='Porter - Other';
UPDATE beer_styles SET bjcp_style_id='30B' WHERE name='Pumpkin / Yam Beer';
UPDATE beer_styles SET bjcp_style_id='31A' WHERE name='Rye Beer';
UPDATE beer_styles SET bjcp_style_id='22B' WHERE name='Rye Wine';
UPDATE beer_styles SET bjcp_style_id='17C' WHERE name='Scotch Ale / Wee Heavy';
UPDATE beer_styles SET bjcp_style_id='32B' WHERE name='Smoked Beer';
UPDATE beer_styles SET bjcp_style_id='23C' WHERE name='Sour - Flanders Oud Bruin';
UPDATE beer_styles SET bjcp_style_id='23B' WHERE name='Sour - Flanders Red Ale';
UPDATE beer_styles SET bjcp_style_id='28B' WHERE name='Sour - Fruited';
UPDATE beer_styles SET bjcp_style_id='23G' WHERE name='Sour - Fruited Gose';
UPDATE beer_styles SET bjcp_style_id='28C' WHERE name='Sour - Other';
UPDATE beer_styles SET bjcp_style_id='16D' WHERE name='Stout - Belgian';
UPDATE beer_styles SET bjcp_style_id='16B' WHERE name='Stout - Coffee';
UPDATE beer_styles SET bjcp_style_id='16D' WHERE name='Stout - Foreign / Export';
UPDATE beer_styles SET bjcp_style_id='20C' WHERE name='Stout - Imperial / Double';
UPDATE beer_styles SET bjcp_style_id='20C' WHERE name='Stout - Imperial / Double Coffee';
UPDATE beer_styles SET bjcp_style_id='16A' WHERE name='Stout - Imperial / Double Milk';
UPDATE beer_styles SET bjcp_style_id='16B' WHERE name='Stout - Imperial / Double Oatmeal';
UPDATE beer_styles SET bjcp_style_id='20C' WHERE name='Stout - Imperial / Double Pastry';
UPDATE beer_styles SET bjcp_style_id='15B' WHERE name='Stout - Irish Dry';
UPDATE beer_styles SET bjcp_style_id='16A' WHERE name='Stout - Milk / Sweet';
UPDATE beer_styles SET bjcp_style_id='16B' WHERE name='Stout - Oatmeal';
UPDATE beer_styles SET bjcp_style_id='20B' WHERE name='Stout - Other';
UPDATE beer_styles SET bjcp_style_id='16A' WHERE name='Stout - Pastry';
UPDATE beer_styles SET bjcp_style_id='20C' WHERE name='Stout - Russian Imperial';
UPDATE beer_styles SET bjcp_style_id='22B' WHERE name='Strong Ale - American';
UPDATE beer_styles SET bjcp_style_id='17A' WHERE name='Strong Ale - English';
UPDATE beer_styles SET bjcp_style_id='22B' WHERE name='Strong Ale - Other';
UPDATE beer_styles SET bjcp_style_id='17B' WHERE name='Traditional Ale';
UPDATE beer_styles SET bjcp_style_id='28B' WHERE name='Wild Ale - American';
UPDATE beer_styles SET bjcp_style_id='28C' WHERE name='Wild Ale - Other';
UPDATE beer_styles SET bjcp_style_id='30C' WHERE name='Winter Ale';