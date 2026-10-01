# 03-estados.py | spec secao 3, F0-035 | idempotente; com MS_REVERTER=1 desfaz
import os
from django.template.defaultfilters import slugify
from plane.db.models import Project, State

REVERTER = os.environ.get("MS_REVERTER") == "1"
NOMES = {
    "unstarted": ("Todo", "A fazer"),
    "started": ("In Progress", "Em andamento"),
    "completed": ("Done", "Concluída"),
    "cancelled": ("Cancelled", "Cancelada"),
    "triage": ("Triage", "Triagem"),
}
projeto = Project.objects.get(workspace__slug="marketing-suite", identifier="MS")
for s in State.all_state_objects.filter(project=projeto, deleted_at__isnull=True).order_by("sequence"):
    par = NOMES.get(s.group)
    if par is None:
        print("MANTIDO", s.group, s.name)
        continue
    de, para = (par[1], par[0]) if REVERTER else par
    if s.name != de:
        print("MANTIDO", s.group, s.name)
        continue
    if State.all_state_objects.filter(project=projeto, name=para, deleted_at__isnull=True).exists():
        print("CONFLITO", s.group, s.name, "->", para, "ja existe; nada feito")
        continue
    State.all_state_objects.filter(id=s.id).update(name=para, slug=slugify(para))
    print("RENOMEADO", s.group, s.name, "->", para)
