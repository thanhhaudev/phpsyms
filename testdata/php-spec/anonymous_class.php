<?php

namespace Test;

class Factory
{
    public function make()
    {
        return new class {
            public function hello(): string
            {
                return "hi";
            }
        };
    }
}
