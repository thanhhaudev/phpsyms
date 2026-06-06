<?php

namespace Test;

class NullCoalesceCases
{
    public function basic(?array $data): string
    {
        return $data['key'] ?? 'default';
    }

    public function chained(?array $a, ?array $b): string
    {
        return $a['x'] ?? $b['y'] ?? 'fallback';
    }
}
