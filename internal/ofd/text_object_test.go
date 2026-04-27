package ofd

import (
	"testing"
)

func TestParseTextObjects_HelloWorld(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ofd:Page xmlns:ofd="http://www.ofdspec.org/2016">
  <ofd:Content>
    <ofd:Layer ID="2">
      <ofd:TextObject ID="4" Boundary="31.7 25.4 40.5 5" Font="3" Size="3.0">
        <ofd:TextCode X="0" Y="3" DeltaX="3 3 3 3 1.5 1.5 1.5 1.5 1.5 1.5 1.5 1.5 1.5 1.5 1.5 1.5 1.5 1.5 1.5 1.5 1.5">你好呀，OFD Reader&amp;Writer！</ofd:TextCode>
      </ofd:TextObject>
    </ofd:Layer>
  </ofd:Content>
</ofd:Page>`)

	objects, err := parseTextObjects(data)
	if err != nil {
		t.Fatalf("parseTextObjects: %v", err)
	}

	if len(objects) != 1 {
		t.Fatalf("expected 1 TextObject, got %d", len(objects))
	}

	obj := objects[0]
	if obj.ID != 4 {
		t.Errorf("TextObject ID = %d, want 4", obj.ID)
	}
	if obj.Font != 3 {
		t.Errorf("TextObject Font = %d, want 3", obj.Font)
	}
	if obj.Size != 3.0 {
		t.Errorf("TextObject Size = %f, want 3.0", obj.Size)
	}
	if len(obj.Boundary) != 4 {
		t.Errorf("Boundary len = %d, want 4", len(obj.Boundary))
	}
	if len(obj.Codes) != 1 {
		t.Fatalf("expected 1 TextCode, got %d", len(obj.Codes))
	}

	code := obj.Codes[0]
	if code.X != 0 {
		t.Errorf("TextCode X = %f, want 0", code.X)
	}
	if code.Y != 3 {
		t.Errorf("TextCode Y = %f, want 3", code.Y)
	}
	if code.Size != 3.0 {
		t.Errorf("TextCode Size = %f, want 3.0", code.Size)
	}
	expectedText := "你好呀，OFD Reader&Writer！"
	if code.Text != expectedText {
		t.Errorf("TextCode Text = %q, want %q", code.Text, expectedText)
	}
}

func TestParseTextObjects_MultipleTextCodes(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ofd:Content xmlns:ofd="http://www.ofdspec.org/2016">
  <ofd:Page>
    <ofd:Layer>
      <ofd:TextObject ID="1" Font="5" Size="4">
        <ofd:TextCode X="0" Y="0">First</ofd:TextCode>
        <ofd:TextCode X="10" Y="0">Second</ofd:TextCode>
      </ofd:TextObject>
    </ofd:Layer>
  </ofd:Page>
</ofd:Content>`)

	objects, err := parseTextObjects(data)
	if err != nil {
		t.Fatalf("parseTextObjects: %v", err)
	}

	if len(objects) != 1 {
		t.Fatalf("expected 1 TextObject, got %d", len(objects))
	}
	obj := objects[0]
	if len(obj.Codes) != 2 {
		t.Fatalf("expected 2 TextCodes, got %d", len(obj.Codes))
	}
	if obj.Codes[0].Text != "First" {
		t.Errorf("first TextCode = %q, want %q", obj.Codes[0].Text, "First")
	}
	if obj.Codes[1].Text != "Second" {
		t.Errorf("second TextCode = %q, want %q", obj.Codes[1].Text, "Second")
	}
}

func TestParseTextObjects_DeltaXWithG(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ofd:Content xmlns:ofd="http://www.ofdspec.org/2016">
  <ofd:Page>
    <ofd:TextObject ID="5" Font="3" Size="2.5">
      <ofd:TextCode X="0" Y="5" DeltaX="g 10 4.0 2.5 2.5 g 5 3.0">测试文本</ofd:TextCode>
    </ofd:TextObject>
  </ofd:Page>
</ofd:Content>`)

	objects, err := parseTextObjects(data)
	if err != nil {
		t.Fatalf("parseTextObjects: %v", err)
	}

	if len(objects) != 1 {
		t.Fatalf("expected 1 TextObject, got %d", len(objects))
	}
	code := objects[0].Codes[0]
	if len(code.DeltaX) == 0 {
		t.Error("DeltaX should not be empty")
	}
	nonGCount := 0
	for _, d := range code.DeltaX {
		if d != 0 {
			nonGCount++
		}
	}
	if nonGCount == 0 {
		t.Error("DeltaX should contain numeric values beyond 'g' placeholders")
	}
}

func TestParseTextObjects_MultipleObjects(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ofd:Content xmlns:ofd="http://www.ofdspec.org/2016">
  <ofd:Page>
    <ofd:TextObject ID="1" Font="3" Size="3.0"><ofd:TextCode X="0" Y="0">Alpha</ofd:TextCode></ofd:TextObject>
    <ofd:TextObject ID="2" Font="3" Size="3.0"><ofd:TextCode X="0" Y="5">Beta</ofd:TextCode></ofd:TextObject>
  </ofd:Page>
</ofd:Content>`)

	objects, err := parseTextObjects(data)
	if err != nil {
		t.Fatalf("parseTextObjects: %v", err)
	}

	if len(objects) != 2 {
		t.Fatalf("expected 2 TextObjects, got %d", len(objects))
	}
	if objects[0].ID != 1 || objects[0].Codes[0].Text != "Alpha" {
		t.Errorf("first object: ID=%d Text=%q", objects[0].ID, objects[0].Codes[0].Text)
	}
	if objects[1].ID != 2 || objects[1].Codes[0].Text != "Beta" {
		t.Errorf("second object: ID=%d Text=%q", objects[1].ID, objects[1].Codes[0].Text)
	}
}

func TestParseTextObjects_Direction(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ofd:Content xmlns:ofd="http://www.ofdspec.org/2016">
  <ofd:Page>
    <ofd:TextObject ID="1" Font="3" Size="4" Direction="90">
      <ofd:TextCode X="0" Y="0">Vertical</ofd:TextCode>
    </ofd:TextObject>
  </ofd:Page>
</ofd:Content>`)

	objects, err := parseTextObjects(data)
	if err != nil {
		t.Fatalf("parseTextObjects: %v", err)
	}

	if len(objects) != 1 {
		t.Fatalf("expected 1 TextObject, got %d", len(objects))
	}
	if objects[0].Direction != "90" {
		t.Errorf("Direction = %q, want %q", objects[0].Direction, "90")
	}
}

func TestParseTextObjects_NoTextObject(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ofd:Page xmlns:ofd="http://www.ofdspec.org/2016">
  <ofd:ImageObject ID="1"/>
</ofd:Page>`)

	objects, err := parseTextObjects(data)
	if err != nil {
		t.Fatalf("parseTextObjects: %v", err)
	}
	if len(objects) != 0 {
		t.Errorf("expected 0 objects, got %d", len(objects))
	}
}

func TestParseTextObjects_EmptyDocument(t *testing.T) {
	objects, err := parseTextObjects([]byte(`<?xml version="1.0"?><empty/>`))
	if err != nil {
		t.Fatalf("parseTextObjects: %v", err)
	}
	if len(objects) != 0 {
		t.Errorf("expected 0 objects, got %d", len(objects))
	}
}

func TestTextCode_String(t *testing.T) {
	code := TextCode{
		Text:   "Hello",
		X:      10.5,
		Y:      20.5,
		DeltaX: []float64{1.0, 2.0},
		DeltaY: []float64{0.5},
		Size:   4.0,
	}
	_ = code
}