<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Observability;

class MetricsCollector
{
    /** @var array<string, int> */
    private array $counters = [];
    
    /** @var array<string, float> */
    private array $gauges = [];
    
    /** @var array<string, array<float>> */
    private array $histograms = [];

    public function incrementCounter(string $name, array $labels = []): void
    {
        $key = $this->buildKey($name, $labels);
        $this->counters[$key] = ($this->counters[$key] ?? 0) + 1;
    }

    public function setGauge(string $name, float $value, array $labels = []): void
    {
        $key = $this->buildKey($name, $labels);
        $this->gauges[$key] = $value;
    }

    public function observeHistogram(string $name, float $value, array $labels = []): void
    {
        $key = $this->buildKey($name, $labels);
        if (!isset($this->histograms[$key])) {
            $this->histograms[$key] = [];
        }
        $this->histograms[$key][] = $value;
    }

    public function getCounter(string $name, array $labels = []): int
    {
        $key = $this->buildKey($name, $labels);
        return $this->counters[$key] ?? 0;
    }

    public function getGauge(string $name, array $labels = []): float
    {
        $key = $this->buildKey($name, $labels);
        return $this->gauges[$key] ?? 0.0;
    }

    public function getHistogramValues(string $name, array $labels = []): array
    {
        $key = $this->buildKey($name, $labels);
        return $this->histograms[$key] ?? [];
    }

    public function toPrometheusFormat(): string
    {
        $output = [];

        foreach ($this->counters as $key => $value) {
            $output[] = "# TYPE {$key} counter";
            $output[] = "{$key} {$value}";
        }

        foreach ($this->gauges as $key => $value) {
            $output[] = "# TYPE {$key} gauge";
            $output[] = "{$key} {$value}";
        }

        foreach ($this->histograms as $key => $values) {
            if (empty($values)) {
                continue;
            }
            
            $count = count($values);
            $sum = array_sum($values);
            
            $output[] = "# TYPE {$key} histogram";
            $output[] = "{$key}_count {$count}";
            $output[] = "{$key}_sum {$sum}";
        }

        return implode("\n", $output);
    }

    private function buildKey(string $name, array $labels): string
    {
        if (empty($labels)) {
            return $name;
        }

        $labelParts = [];
        foreach ($labels as $key => $value) {
            $labelParts[] = "{$key}=\"{$value}\"";
        }

        return $name . '{' . implode(',', $labelParts) . '}';
    }

    public function reset(): void
    {
        $this->counters = [];
        $this->gauges = [];
        $this->histograms = [];
    }
}
