<?php

namespace Test;

class InterpolationCases
{
    public function simple()
    {
        $name = "world";
        return "hello $name";
    }

    public function braced()
    {
        $obj = new \stdClass();
        $obj->prop = "x";
        return "value: {$obj->prop}";
    }

    public function dollarBraced()
    {
        $name = "world";
        return "hello ${name}";
    }

    public function arrayAccess()
    {
        $arr = ['k' => 'v'];
        return "val: {$arr['k']}";
    }
}
