<?php

namespace Test;

class HeredocCases
{
    public function basic()
    {
        $name = "world";
        return <<<EOT
hello $name end
EOT;
    }

    public function nowdoc()
    {
        $name = "world";
        return <<<'EOT'
hello $name end
EOT;
    }

    public function indented()
    {
        return <<<EOT
        indented content
        EOT;
    }
}
