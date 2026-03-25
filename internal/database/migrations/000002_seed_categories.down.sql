INSERT INTO categories (name, type, icon, applicable_to_freq, created_at, updated_at) VALUES
    ('Primary Income',        'income',  '💰', 'fixed',    now(), now()),
    ('Side Income',           'income',  '💼', 'variable', now(), now()),
    ('Investments',           'income',  '📈', 'variable', now(), now()),
    ('Other Income',          'income',  '🎁', 'variable', now(), now()),
    ('Housing & Utilities',   'outcome', '🏠', 'fixed',    now(), now()),
    ('Food & Groceries',      'outcome', '🍽️', 'variable', now(), now()),
    ('Transportation',        'outcome', '🚗', 'variable', now(), now()),
    ('Health & Fitness',      'outcome', '🏥', 'variable', now(), now()),
    ('Education',             'outcome', '📚', 'variable', now(), now()),
    ('Entertainment & Travel','outcome', '🎉', 'variable', now(), now()),
    ('Taxes & Fees',          'outcome', '📋', 'fixed',    now(), now()),
    ('Pets & Misc',           'outcome', '🐾', 'variable', now(), now());
